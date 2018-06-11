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

	"github.com/kaushiksriram100/ansible-deployer-tp/deployer"
)

const OO_API_TOKEN = "xzRatZ4Spz8j644ZRxxb"

func main() {

	//XL oneops machines have 8 logical cores. using 6.
	runtime.GOMAXPROCS(6)

	inputs := &deployer.Inputs{}

	//log file path
	(*inputs).Log_file_path = flag.String("logfile", "/var/tmp/ansible-deployer/log/", "--logfile=logfile path. Default is /var/tmp/ansible-deployer/log/")

	// other inputs
	(*inputs).Ansible_playbook_path = flag.String("playbookpath", "/opt/tp-deployer/", "--playbookpath <fullpath of playbook>")
	(*inputs).Ansible_playbook_action = flag.String("playbookaction", "main.yml", "--playbookaction <main.yml, start.yml, stop.yml>. default main.yml")
	(*inputs).Oneops_jar_path = flag.String("invjar", "/opt/tp-deployer/oneops-inventory/oo-wrapper.py", "--invjar <fullpath of oneops inv jar. check documentation>")
	(*inputs).Ansible_tags = flag.String("tags", "", "--tags start")
	(*inputs).Ansbile_skip_tags = flag.String("skip-tags", "", "--skip-tags start")
	(*inputs).Target_type = flag.String("targettype", "inventories", "--targettype oneops or physical. This the first directory in conf hierarchy")
	(*inputs).Ansible_user = flag.String("ansibleuser", "stream-splitter", "--ansibleuser <username to ssh>")
	(*inputs).Hosts_limit = flag.String("hostlimit", "all", "--hostlimit <all>")
	(*inputs).Max_fail_percentage = flag.String("mfp", "100", "this is the ansible max_fail_percentage value. --mfp 100")
	(*inputs).S1 = flag.String("s1", "100%", "percent or total hosts to run the playbook parallely in one pass. --s1 10")
	(*inputs).S2 = flag.String("s2", "100%", "percent or total hosts to run the playbook parallely in one pass. --s2 100")
	(*inputs).S3 = flag.String("s3", "100%", "percent or total hosts to run the playbook parallely in one pass. --s3 10")

	flag.Parse()

	//Create log file or error out if unable to create. This return logfile pointer will be used to set the log file path for log.SetOutput

	logfile, err := deployer.CreateLogFile(inputs.Log_file_path)

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
	EnvVarMap := deployer.ExtractEnvVars(inputs, logfile)
	//	fmt.Println(EnvVarMap)

	//YADA-YADA!!! At this point I have got everything that I need to run my deployment. Call the deploy function and Booooooo (not to confuse with boo template.. just boooooo)!

	if len(EnvVarMap) == 0 {
		log.Fatal("ERROR: Couldn't properly collect details from env config file. Check if files are consistent and no permission issues")
		return

	}

	//call Deploy. Pass the log file handler so that package is able to update the log file. You need to setoutput(logfile) in each function in package
	// I am not using any return vairable. But good to have. Most info are populated in the log file.

	deployer.Deploy(&EnvVarMap, inputs, logfile, OO_API_TOKEN)

	return

}
