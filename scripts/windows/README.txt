Gitra for Windows x64

1. Install Git for Windows from https://gitforwindows.org/ using its default
   Git Credential Manager and command-line PATH options. Reopen your terminal
   after installation. Go and WSL are not required.
2. Extract the entire ZIP, then double-click Gitra.cmd.
3. Use the arrow keys and Enter to navigate; Esc returns, Q quits.
   The terminal interface currently uses Chinese labels.
4. For GitHub browser authorization, install GitHub CLI (gh) from
   https://cli.github.com/. You can instead paste an access token without gh.
5. Bind a project folder to an account. Regular folders can be initialized
   during binding. Create a local commit before the first upload (U).

HTTPS credentials use the configured Git credential helper, normally Git
Credential Manager with Windows Credential Manager. SSH keys with passphrases
need an SSH agent. gitra.exe also supports command-line use, e.g. --version.

If an error occurs, the launcher keeps the window open so you can read it.
This is a portable terminal application, not a graphical desktop application.
Browser authorization and interactive desktop rendering require a manual trial.
