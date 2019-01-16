# goversionflag

goversionflag is an go package aiming to help identifying versions of deployed go binarys.

## Installation
```
go get -u github.com/stockholmuniversity/goversionflag
```

## Usage
```go
package main

import (
        "fmt"
        "github.com/stockholmuniversity/goversionflag"
)

func main() {
        goversionflag.PrintVersionAndExit()
        fmt.Println("End of program, will not be shown when run with '-version'")
}
```

## Build your program
When building your program with goversionflag you must add buildflags in the build process:
```
go build -ldflags "-X github.com/stockholmuniversity/goversionflag.projectName=$project -X github.com/stockholmuniversity/goversionflag.gitCommit=$commit -X github.com/stockholmuniversity/goversionflag.buildTime=$time"
```
For more information see documentation and comments in printVersion.go

