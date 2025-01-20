#!/bin/bash

# Define the license notice
LICENSE_NOTICE="// Copyright (c) 2022-present, DiceDB contributors
// All rights reserved. Licensed under the BSD 3-Clause License. See LICENSE file in the project root for full license information."

# Find all Go files and add the notice if not already present
find . -type f -name "*.go" | while read -r FILE; do
    # Check if the file already contains the license notice
    if ! grep -q "$LICENSE_NOTICE" "$FILE"; then
        # Add the license notice at the top of the file
        echo -e "$LICENSE_NOTICE\n\n$(cat "$FILE")" > "$FILE"
        echo "Added license notice to $FILE"
    else
        echo "License notice already present in $FILE"
    fi
done

echo "Finished adding license notice to all Go files."
