#!/bin/bash

echo "Creating materialized views..."

# Set the directory path
directory="./"

# Prompt for PostgreSQL username
read -p "Enter PostgreSQL username: " username

# Prompt for PostgreSQL password (without displaying it on the console)
read -s -p "Enter PostgreSQL password: " password
echo

# Prompt for PostgreSQL database name
read -p "Enter PostgreSQL database name: " database_name

# Loop through each .sql file starting with "drop_" in the directory
for file in "$directory"/*.sql; do
    if [ -f "$file" ]; then
        # Extract the file name without extension
        filename=$(basename "$file" .sql)

        # Run the SQL file using psql with username, password, and database name
        PGPASSWORD="$password" psql -U "$username" -d "$database_name" -f "$file"

        # Check the exit status of psql
        if [ $? -eq 0 ]; then
            echo "Successfully executed $filename.sql"
        else
            echo "Error executing $filename.sql"
        fi
    fi
done

