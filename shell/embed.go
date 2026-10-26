package shell

import _ "embed"

const IntegrationVersion = 30

//go:embed shellux.zsh
var Zsh []byte

//go:embed shellux.bash
var Bash []byte
