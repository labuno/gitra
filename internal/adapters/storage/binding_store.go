package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zhanhd/gitra/internal/domain"
)

// BindingStore implements ports.BindingStore with a single bindings.json
// document (baseline §27).
type BindingStore struct {
	dir string
}

// NewBindingStore roots the store at the gitra config directory.
func NewBindingStore(dir string) *BindingStore { return &BindingStore{dir: dir} }

type bindingFile struct {
	ID         string `json:"id"`
	Repository struct {
		Path string `json:"path"`
	} `json:"repository"`
	AccountID string `json:"account_id"`
	Strategy  string `json:"strategy"`
	Revision  int    `json:"revision"`
}

type bindingsDocument struct {
	SchemaVersion int           `json:"schema_version"`
	Bindings      []bindingFile `json:"bindings"`
}

func (s *BindingStore) path() string { return filepath.Join(s.dir, "bindings.json") }

func (s *BindingStore) load() (bindingsDocument, error) {
	doc := bindingsDocument{SchemaVersion: schemaVersion}
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return doc, err
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("bindings.json: %w", err)
	}
	if doc.SchemaVersion != schemaVersion {
		return doc, fmt.Errorf("unsupported bindings schema_version %d", doc.SchemaVersion)
	}
	return doc, nil
}

func (s *BindingStore) save(doc bindingsDocument) error {
	doc.SchemaVersion = schemaVersion
	if doc.Bindings == nil {
		doc.Bindings = []bindingFile{}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(s.path(), append(data, '\n'), 0o600)
}

func toBindingFile(b domain.RepositoryBinding) bindingFile {
	var file bindingFile
	file.ID = string(b.ID)
	file.Repository.Path = b.Repository.Path
	file.AccountID = string(b.AccountID)
	file.Strategy = b.Strategy
	file.Revision = b.Revision
	return file
}

func (f bindingFile) toDomain() (domain.RepositoryBinding, error) {
	binding := domain.RepositoryBinding{
		ID:         domain.BindingID(f.ID),
		AccountID:  domain.AccountID(f.AccountID),
		Repository: domain.RepositoryRef{Path: f.Repository.Path},
		Strategy:   f.Strategy,
		Revision:   f.Revision,
	}
	if err := binding.Validate(); err != nil {
		return domain.RepositoryBinding{}, fmt.Errorf("binding %s is invalid: %w", f.ID, err)
	}
	return binding, nil
}

// Save inserts or replaces one binding record.
func (s *BindingStore) Save(ctx context.Context, binding domain.RepositoryBinding) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	doc, err := s.load()
	if err != nil {
		return err
	}
	replaced := false
	for i, existing := range doc.Bindings {
		if existing.ID == string(binding.ID) {
			doc.Bindings[i] = toBindingFile(binding)
			replaced = true
			continue
		}
		if existing.Repository.Path == binding.Repository.Path {
			return fmt.Errorf("%w: %s", domain.ErrAlreadyBound, binding.Repository.Path)
		}
	}
	if !replaced {
		doc.Bindings = append(doc.Bindings, toBindingFile(binding))
	}
	return s.save(doc)
}

// Get returns one binding by ID.
func (s *BindingStore) Get(ctx context.Context, id domain.BindingID) (domain.RepositoryBinding, error) {
	doc, err := s.load()
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	for _, file := range doc.Bindings {
		if file.ID == string(id) {
			return file.toDomain()
		}
	}
	return domain.RepositoryBinding{}, fmt.Errorf("%w: %s", domain.ErrBindingNotFound, id)
}

// FindByRepository looks up a binding by canonical repository path.
func (s *BindingStore) FindByRepository(ctx context.Context, canonicalPath string) (domain.RepositoryBinding, bool, error) {
	doc, err := s.load()
	if err != nil {
		return domain.RepositoryBinding{}, false, err
	}
	for _, file := range doc.Bindings {
		if file.Repository.Path == canonicalPath {
			binding, err := file.toDomain()
			if err != nil {
				return domain.RepositoryBinding{}, false, err
			}
			return binding, true, nil
		}
	}
	return domain.RepositoryBinding{}, false, nil
}

// ListByAccount returns every binding referencing the account.
func (s *BindingStore) ListByAccount(ctx context.Context, id domain.AccountID) ([]domain.RepositoryBinding, error) {
	doc, err := s.load()
	if err != nil {
		return nil, err
	}
	var bindings []domain.RepositoryBinding
	for _, file := range doc.Bindings {
		if file.AccountID != string(id) {
			continue
		}
		binding, err := file.toDomain()
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

// Delete removes one binding record.
func (s *BindingStore) Delete(ctx context.Context, id domain.BindingID) error {
	doc, err := s.load()
	if err != nil {
		return err
	}
	kept := doc.Bindings[:0]
	found := false
	for _, file := range doc.Bindings {
		if file.ID == string(id) {
			found = true
			continue
		}
		kept = append(kept, file)
	}
	if !found {
		return fmt.Errorf("%w: %s", domain.ErrBindingNotFound, id)
	}
	doc.Bindings = kept
	return s.save(doc)
}
