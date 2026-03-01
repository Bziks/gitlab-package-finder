# oapi-codegen

Source: https://github.com/deepmap/oapi-codegen

## Installation

1. Install the library
    ```bash
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
    ```
2. Check the lib
    ```bash
    oapi-codegen --help
    ```
   
If you got the error like: `oapi-codegen command not found`, then add a `go` folder to $PATH.

Example for `zsh`:
1. Edit `.zshrc` file
    ```bash
    nvim ~/.zshrc
    ```
2. Scroll to the end of file and append the next string
    ```bash
    export PATH="$PATH:/home/YOUR_USER/go/bin"
    ```