
mkdir -p ~/.local/bin

version=v0.1
arch=$(uname -sm)
case "$arch" in
    Linux\ *64) wget -O ~/.local/bin/binch https://github.com/tunz/binch-go/releases/download/${version}/binch-linux-x64  ;;
    Darwin\ *64) wget -O ~/.local/bin/binch https://github.com/tunz/binch-go/releases/download/${version}/binch-macos  ;;
    *) echo "This OS is not supported yet. Please report us in https://github.com/tunz/binch-go"; exit 1;;
esac
chmod +x ~/.local/bin/binch

if [[ "$PATH" != *"$HOME/.local/bin:"* && "$PATH" != *":$HOME/.local/bin" ]]; then
    if [[ "$SHELL" =~ "bash" ]]; then
            echo "" >> ~/.bashrc
            echo 'export PATH=~/.local/bin:$PATH' >> ~/.bashrc
    elif [[ "$SHELL" =~ "zsh"  ]]; then
        echo "" >> ~/.zsrrc
        echo 'export PATH=~/.local/bin:$PATH' >> ~/.zshrc
    else
        echo "Please add ~/.local/bin to your PATH"
        echo ""
        echo '    export PATH=~/.local/bin:$PATH'
        echo ""
    fi
fi

echo "Installation Completed!"
