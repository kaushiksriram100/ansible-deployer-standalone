/* PURPOSE: To create a worker which can run the ansible-playbook command for each env/host in oneops.
This main function depends on deployer package.
AUTHOR: SRIRAM KAUSHIK
*/

// To do : add -h helper.

package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/kaushiksriram100/ansible-deployer-standalone/deployer"
)

const OO_API_TOKEN = "<sorry>"

func main() {

	//XL oneops machines have 8 logical cores.
	runtime.GOMAXPROCS(5)

	//Get logfile path and config file path from arguments.
	var log_file_path = flag.String("logfile", "/var/tmp/ansible-deployer/log/", "--logfile=logfile path. Default is /var/tmp/ansible-deployer/log/")

	//Get the ansible playbook file and oneops-inventory jar path.

	var ansible_playbook_path = flag.String("playbookpath", "/Users/skaush1/Documents/my_dev_env/ansible-workspace/wm-splunk-universal-forwarder/", "--playbookpath <fullpath of playbook>")
	var ansible_playbook_action = flag.String("playbookaction", "main.yml", "--playbookaction <main.yml, start.yml, stop.yml>. default main.yml")
	var oneops_jar_path = flag.String("invjar", "/Users/skaush1/Documents/my_dev_env/oneops-inventory/oo-wrapper.py", "--invjar <fullpath of oneops inv jar. check documentation>")
	var target_type = flag.String("targettype", "oneops", "--targettype oneops or physical. This the first directory in conf hierarchy")

	flag.Parse()

	//Create log file or error out if unable to create. This return logfile pointer will be used to set the log file path for log.SetOutput

	logfile, err := deployer.CreateLogFile(log_file_path)

	if err != nil {
		fmt.Println("Error Occured while creating log file:", err)
		return
	}

	//need to set the output
	log.SetOutput(logfile)

	//This will avoid any memory leaks when the program ends.
	defer logfile.Close()

	//At this points we have the log files in place. Now let's start some heavy lifting.

	//This function will read all the files and extract all variables and populate the EnvVarMap map with data. We pass the logfile handler so that our go routines are able to update the logfile
	EnvVarMap := deployer.ExtractEnvVars(target_type, ansible_playbook_path, logfile)
	//	fmt.Println(EnvVarMap)

	//YADA-YADA!!! At this point I have got everything that I need to run my deployment. Call the deploy function and Booooooo (not to confuse with boo template.. just boooooo)!

	if len(EnvVarMap) == 0 {
		log.Fatal("ERROR: Couldn't properly collect details from env config file. Check if files are consistent and no permission issues")
		return

	}

	//call Deploy. Pass the log file handler so that package is able to update the log file. You need to setoutput(logfile) in each function in package
	// I am not using any return vairable. But good to have. Most info are populated in the log file.
	deployer.Deploy(&EnvVarMap, ansible_playbook_path, ansible_playbook_action, oneops_jar_path, logfile, log_file_path, OO_API_TOKEN)

	return

}
