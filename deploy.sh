#!/bin/bash

# DS2API Deploy Script
# Usage: ./deploy.sh

set -e

echo "================================"
echo "DS2API Deployment Script"
echo "================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root (sudo ./deploy.sh)${NC}"
    exit 1
fi

# Get actual user (not root)
ACTUAL_USER=${SUDO_USER:-$USER}
USER_HOME=$(eval echo ~$ACTUAL_USER)

echo -e "${GREEN}Step 1: Installing dependencies...${NC}"

# Update system
apt update && apt upgrade -y

# Install Git
if ! command -v git &> /dev/null; then
    echo "Installing Git..."
    apt install git -y
else
    echo "Git already installed: $(git --version)"
fi

# Install Go
if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    wget -q https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    rm go1.21.6.linux-amd64.tar.gz
else
    echo "Go already installed: $(go version)"
fi

# Install Node.js
if ! command -v node &> /dev/null; then
    echo "Installing Node.js..."
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
    apt install -y nodejs
else
    echo "Node.js already installed: $(node --version)"
fi

echo ""
echo -e "${GREEN}Step 2: Cloning repository...${NC}"

# Clone repo
REPO_DIR="$USER_HOME/ds2api"
if [ -d "$REPO_DIR" ]; then
    echo "Repository already exists at $REPO_DIR"
    read -p "Do you want to pull latest changes? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$REPO_DIR"
        sudo -u $ACTUAL_USER git pull origin custom-vibecode
    fi
else
    cd "$USER_HOME"
    sudo -u $ACTUAL_USER git clone https://github.com/CJackHwang/ds2api.git
    cd "$REPO_DIR"
    sudo -u $ACTUAL_USER git checkout custom-vibecode
fi

echo ""
echo -e "${GREEN}Step 3: Building application...${NC}"

cd "$REPO_DIR"

# Build frontend
echo "Building frontend..."
cd webui
sudo -u $ACTUAL_USER npm install
sudo -u $ACTUAL_USER npm run build
cd ..

# Build backend
echo "Building backend..."
sudo -u $ACTUAL_USER /usr/local/go/bin/go build -o ds2api ./cmd/ds2api

echo ""
echo -e "${GREEN}Step 4: Setting up configuration...${NC}"

# Check if config.json exists
if [ ! -f "$REPO_DIR/config.json" ]; then
    echo -e "${YELLOW}config.json not found. Please create it manually.${NC}"
    echo "Example: cp config.json.example config.json && nano config.json"
fi

# Set admin password
read -p "Enter admin password (or press Enter to skip): " ADMIN_PASS
if [ ! -z "$ADMIN_PASS" ]; then
    echo "export DS2API_ADMIN_KEY=\"$ADMIN_PASS\"" >> /etc/profile
    export DS2API_ADMIN_KEY="$ADMIN_PASS"
    echo -e "${GREEN}Admin password set!${NC}"
fi

echo ""
echo -e "${GREEN}Step 5: Installing systemd service...${NC}"

# Copy service file
cp "$REPO_DIR/ds2api.service" /etc/systemd/system/ds2api.service

# Update service file with actual paths
sed -i "s|/root/ds2api|$REPO_DIR|g" /etc/systemd/system/ds2api.service
sed -i "s|User=root|User=$ACTUAL_USER|g" /etc/systemd/system/ds2api.service
sed -i "s|Group=root|Group=$ACTUAL_USER|g" /etc/systemd/system/ds2api.service

# Update admin password in service file
if [ ! -z "$ADMIN_PASS" ]; then
    sed -i "s|CHANGE_THIS_PASSWORD|$ADMIN_PASS|g" /etc/systemd/system/ds2api.service
fi

# Reload systemd
systemctl daemon-reload
systemctl enable ds2api

echo ""
echo -e "${GREEN}Step 6: Starting service...${NC}"

systemctl start ds2api
sleep 2

# Check status
if systemctl is-active --quiet ds2api; then
    echo -e "${GREEN}✓ DS2API service started successfully!${NC}"
else
    echo -e "${RED}✗ Failed to start DS2API service${NC}"
    echo "Check logs: journalctl -u ds2api -n 50"
    exit 1
fi

echo ""
echo -e "${GREEN}Step 7: Installing Cloudflare Tunnel (optional)...${NC}"
read -p "Do you want to install Cloudflare Tunnel? (y/n) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Install cloudflared
    if ! command -v cloudflared &> /dev/null; then
        echo "Installing cloudflared..."
        wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
        dpkg -i cloudflared-linux-amd64.deb
        rm cloudflared-linux-amd64.deb
    fi

    echo ""
    echo -e "${YELLOW}To setup Cloudflare Tunnel:${NC}"
    echo "1. Run: cloudflared tunnel login"
    echo "2. Run: cloudflared tunnel create ds2api"
    echo "3. Create config: nano ~/.cloudflared/config.yml"
    echo "4. Run: cloudflared tunnel route dns ds2api yourdomain.com"
    echo "5. Run: cloudflared service install"
    echo ""
    echo "See DEPLOY.md for detailed instructions."
fi

echo ""
echo "================================"
echo -e "${GREEN}Deployment Complete!${NC}"
echo "================================"
echo ""
echo "Service status: $(systemctl is-active ds2api)"
echo "View logs: journalctl -u ds2api -f"
echo "Restart: systemctl restart ds2api"
echo ""
echo "Access admin panel:"
echo "- Local: http://localhost:5001/admin"
echo "- With Cloudflare Tunnel: https://yourdomain.com/admin"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Configure config.json if not done"
echo "2. Setup Cloudflare Tunnel (see DEPLOY.md)"
echo "3. Test the admin panel"
echo "4. Setup firewall (ufw)"
echo ""
