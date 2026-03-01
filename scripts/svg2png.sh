#!/bin/bash
#
# SVG to PNG Converter for HotPlex
# Converts all SVG files in docs-site/public to high-resolution PNG
#
# Usage:
#   ./scripts/svg2png.sh [options]
#
# Options:
#   -z, --zoom         Zoom factor for resolution (default: 4)
#   -b, --background   Background color in hex (default: transparent)
#   -h, --help         Show this help message
#
# Examples:
#   ./scripts/svg2png.sh                    # Convert all with defaults
#   ./scripts/svg2png.sh -z 8               # 8x resolution (8K+)
#   ./scripts/svg2png.sh -b "#FFFFFF"       # White background
#

set -e

# Default values
ZOOM=4
BACKGROUND=""

# Source directories (relative to project root)
IMAGE_DIR="docs-site/public/images"
PUBLIC_DIR="docs-site/public"
GITHUB_ASSETS=".github/assets"

# Colors for output (only if TTY)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    BLUE=''
    NC=''
fi

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -z|--zoom)
            ZOOM="$2"
            shift 2
            ;;
        -b|--background)
            BACKGROUND="$2"
            shift 2
            ;;
        -h|--help)
            sed -n '2,18p' "$0" | sed 's/^# //'
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Check dependencies
check_dependencies() {
    if ! command -v rsvg-convert &> /dev/null; then
        echo -e "${RED}Missing: librsvg${NC}"
        echo "Install: brew install librsvg"
        exit 1
    fi
}

# Convert single SVG to PNG
convert_svg() {
    local svg_file="$1"
    local output_file="$2"
    local filename=$(basename "$svg_file")

    # Build command
    local cmd="rsvg-convert -z $ZOOM"
    [ -n "$BACKGROUND" ] && cmd="$cmd --background-color=\"$BACKGROUND\""
    cmd="$cmd -o \"$output_file\" \"$svg_file\""

    echo -e "  ${BLUE}→${NC} $filename"
    eval $cmd
}

# Main
main() {
    check_dependencies

    local total=0

    # Print header
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  SVG to PNG Converter${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Zoom:       ${BLUE}${ZOOM}x${NC}"
    echo -e "  Background: ${BLUE}${BACKGROUND:-transparent}${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Convert docs-site/public/images/*.svg
    if [ -d "$IMAGE_DIR" ]; then
        echo -e "${BLUE}Converting diagram images...${NC}"
        mkdir -p "${IMAGE_DIR}/png"
        for svg_file in "$IMAGE_DIR"/*.svg; do
            if [ -f "$svg_file" ]; then
                local filename=$(basename "$svg_file" .svg)
                convert_svg "$svg_file" "${IMAGE_DIR}/png/${filename}.png"
                ((total++))
            fi
        done
        echo ""
    fi

    # Convert docs-site/public/*.svg (logo, avatar)
    if [ -d "$PUBLIC_DIR" ]; then
        echo -e "${BLUE}Converting brand assets...${NC}"
        for svg_file in "$PUBLIC_DIR"/*.svg; do
            if [ -f "$svg_file" ]; then
                local filename=$(basename "$svg_file" .svg)
                convert_svg "$svg_file" "${PUBLIC_DIR}/${filename}.png"
                ((total++))
            fi
        done
        echo ""
    fi

    # Convert .github/assets/*.svg (social preview)
    if [ -d "$GITHUB_ASSETS" ]; then
        echo -e "${BLUE}Converting GitHub assets...${NC}"
        for svg_file in "$GITHUB_ASSETS"/*.svg; do
            if [ -f "$svg_file" ]; then
                local filename=$(basename "$svg_file" .svg)
                convert_svg "$svg_file" "${GITHUB_ASSETS}/${filename}.png"
                ((total++))
            fi
        done
        echo ""
    fi

    echo -e "${GREEN}✓ Done! $total files converted${NC}"
    echo ""
}

main
