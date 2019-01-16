/*
Package goversionflag gives the compiled binary a "--version" argument
if compiled with correct -ldflags
(The variables must be replaced with actual variables from the build system)

 go build -ldflags "-X github.com/stockholmuniversity/goversionflag.projectName=goversionflag -X github.com/stockholmuniversity/goversionflag.gitCommit=gitCommit -X github.com/stockholmuniversity/goversionflag.buildTime=1970-01-01"
Primary usage of this packet should be using the function PrintVersionAndExit() which handles
"--version" and "-version" according to GNU Coding standard 4.7.1:
https://www.gnu.org/prep/standards/html_node/_002d_002dversion.html#g_t_002d_002dversion

If the user of "goversionflag" also use the package "flag" for its own flags, the flags must be declared before PrintVersionAndExit() is called.

There is also the possibility to call GetBuildInformation() to just get the build information without printing or exit the program.
*/
package goversionflag

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

// Var that is set at compile time by the build system.
// Build arguments:
/* go build -ldflags "\
   -X github.com/stockholmuniversity/goversionflag.projectName=${PROJECT} \
   -X github.com/stockholmuniversity/goversionflag.gitCommit=${GIT_COMMIT} \
   -X github.com/stockholmuniversity/goversionflag.buildTime=${TIMESTAMP}"
*/
var (
	projectName string
	gitCommit   string
	buildTime   string
)

// Make our own instance of os.Exit. This must be done so the unittest can replace osExit with an
// function of it own. That way the test can verify the function that uses osExit.
var osExit = os.Exit

// The unittest sets fakeFlag, so that the test can run code as if it was run with "--version"
var fakeFlag = false

/*
PrintVersionAndExit prints information of the build and exits

According to GNU coding standard 4.7.1 argument --version to the binary should:
  "The standard --version option should direct the program to print information about its
  name, version, origin and legal status, all on standard output, and then exit successfully
  Other options and arguments should be ignored once this is seen, and the program should not perform its normal function."
https://www.gnu.org/prep/standards/html_node/_002d_002dversion.html#g_t_002d_002dversion
*/
func PrintVersionAndExit() {

	arg := flag.Bool("version", false, "Prints build information and exits.")
	if flag.Parsed() == false {
		flag.Parse()
	}

	if *arg == true || fakeFlag == true {
		buildversion := GetBuildInformation()
		buildSlice := []string{}
		missingBuildInfo := false
		for k, v := range buildversion {
			buildSlice = append(buildSlice, k+": "+v)
			if v == "" {
				missingBuildInfo = true
			}
		}
		sort.Strings(buildSlice)
		for _, v := range buildSlice {
			fmt.Println(v)
		}
		if missingBuildInfo {
			fmt.Println("Do not have complete buildinfo, see documentaion:")
			fmt.Println("\thttps://github.com/stockholmuniversity/goversionflag")
			fmt.Println("\thttps://godoc.org/github.com/stockholmuniversity/goversionflag")
		}

		osExit(0)
	}
}

// GetBuildInformation returns a map with build information from ci build pipe at compile time.
//
// In most cases you should not use this function but instead PrintVersionAndExit().
// This function assume that the main program implements its own variation of PrintVersionAndExit()
func GetBuildInformation() (buildversion map[string]string) {
	buildversion = map[string]string{
		"projectName": projectName,
		"gitCommit":   gitCommit,
		"buildTime":   buildTime,
	}
	return buildversion
}
