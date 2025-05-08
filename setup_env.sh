#!/bin/bash

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

# Check if config file exists
if [ ! -f "env_vars.conf" ]; then
    echo "Error: env_vars.conf not found"
    exit 1
fi

# Read and set each variable from the config file
while IFS='=' read -r VAR_NAME VAR_VALUE || [ -n "$VAR_NAME" ]; do
    # Skip comments and empty lines
    [[ $VAR_NAME =~ ^#.*$ ]] && continue
    [[ -z $VAR_NAME ]] && continue
    
    # Remove any leading/trailing whitespace
    VAR_NAME=$(echo "$VAR_NAME" | xargs)
    VAR_VALUE=$(echo "$VAR_VALUE" | xargs)
    
    # Add to system-wide profile
    echo "export $VAR_NAME=$VAR_VALUE" >> /etc/profile
    
    # Add to system-wide bash profile
    echo "export $VAR_NAME=$VAR_VALUE" >> /etc/bashrc
    
    # Add to system-wide zsh profile
    echo "export $VAR_NAME=$VAR_VALUE" >> /etc/zshrc
    
    # Add to user profiles
    for user_home in /Users/*; do
        if [ -d "$user_home" ]; then
            # Add to .bash_profile
            echo "export $VAR_NAME=$VAR_VALUE" >> "$user_home/.bash_profile"
            
            # Add to .zshrc
            echo "export $VAR_NAME=$VAR_VALUE" >> "$user_home/.zshrc"
            
            # Set proper permissions
            chown $(stat -f "%Su:%Sg" "$user_home") "$user_home/.bash_profile" "$user_home/.zshrc"
        fi
    done
    
    echo "Set $VAR_NAME=$VAR_VALUE"
done < env_vars.conf

echo "All environment variables have been set system-wide"
echo "Please restart your terminal or run 'source ~/.zshrc' (or ~/.bash_profile) to apply changes" 