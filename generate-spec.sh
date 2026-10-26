#!/bin/sh

# script to remove package names from generated Swagger doc
swag init --v3.1
docs_dir="docs"
package_names=$(find . -name "*.go" -exec grep -h "^package " {} \; | awk '{print $2}' | sort | uniq)

for file in "$docs_dir"/*; do
    [ -f "$file" ] || continue
    for package in $package_names; do
        # Use portable sed for Linux and macOS
        if sed --version >/dev/null 2>&1; then
            # GNU sed (Linux)
            sed -i "s/${package}\.//g" "$file"
        else
            # BSD sed (macOS)
            sed -i '' "s/${package}\.//g" "$file"
        fi
    done
done

# Remove "schemes" line from docs.go (not valid in OpenAPI 3)
if sed --version >/dev/null 2>&1; then
    sed -i '/"schemes": /d' "$docs_dir/docs.go"
else
    sed -i '' '/"schemes": /d' "$docs_dir/docs.go"
fi
