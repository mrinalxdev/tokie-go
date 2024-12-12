# Tokei-Go

![](/assets/banner.png)

currently working on the sorting of the code table and skipping the node_modules directory

# V1.0

![](/assets/banner2.png)

Now the supported flags and arguments are

```bash
# Sort by number of files (ascending)
go run main.go --sort "files asc"

# Sort by lines of code (descending)
go run main.go --sort "lines desc"

# Sort by file size (ascending)
go run main.go --sort "size asc"
```

Also to skip the node_modules and other files
```bash
# Skip node_modules directories
go run main.go --skip-node-modules

# Combine with sorting
go run main.go --sort "files desc" --skip-node-modules

# Combine with pattern exclusion
go run main.go --sort "lines desc" --skip-node-modules --exclude "*.json,*.yml"
```


