package template

// ConfigYAML is the lean live starter written by wonderfeed init.
var ConfigYAML = []byte(`# Wonderfeed operator config (secrets via env or OS credential store)
database_url: "${DATABASE_URL}"
parent_auth_key: "${WONDERFEED_PARENT_AUTH_KEY}"

control_plane:
  bind: "127.0.0.1:8080"

provider:
  ytzero:
    base_url: "http://127.0.0.1:3001"
    auth_method: "none"
    auth_password: "${YTZERO_AUTH_PASSWORD}"
    session_cookie: "${YTZERO_SESSION_COOKIE}"
`)

// ConfigExampleYAML is refreshed on every init (documented reference).
var ConfigExampleYAML = []byte(`# Wonderfeed operator config example (no literal secrets)
# Copy to config.yaml or run: wonderfeed init
#
# Discovery order:
#   WONDERFEED_CONFIG or --config  ->  ~/.config/wonderfeed/config.yaml  ->  ./config.yaml
#
# Placeholders expand from the process environment, then the platform credential store
# (macOS Keychain / Windows Credential Manager) using the placeholder name as the account.

database_url: "${DATABASE_URL}"
parent_auth_key: "${WONDERFEED_PARENT_AUTH_KEY}"

control_plane:
  bind: "127.0.0.1:8080"

provider:
  ytzero:
    base_url: "http://127.0.0.1:3001"
    auth_method: "none"
    auth_password: "${YTZERO_AUTH_PASSWORD}"
    session_cookie: "${YTZERO_SESSION_COOKIE}"
`)
