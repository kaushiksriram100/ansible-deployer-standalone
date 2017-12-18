/* PURPOSE: To create a worker which can run the ansible-playbook command for each env/host in oneops.
AUTHOR: SRIRAM KAUSHIK
*/

package deployer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type EnvVars struct {
	OO_ORG      string
	OO_PLATFORM string
	OO_ASSEMBLY string
	OO_ENV      string
}

var EnvVarMap = make(map[string]EnvVars)

func PopulateHash(path string, fp os.FileInfo, err error) error {
	if fp.IsDir() {
		return nil
	}

	if path == "" {
		return nil
	}

	tmp := strings.Split(path, "/")

	tmp_oo_env := tmp[len(tmp)-2]
	tmp_oo_platform := tmp[len(tmp)-3]
	tmp_oo_assembly := tmp[len(tmp)-4]
	tmp_oo_org := tmp[len(tmp)-5]

	key := tmp_oo_org + "_" + tmp_oo_assembly + "_" + tmp_oo_platform + "_" + tmp_oo_env

	//This hash generates a key which is the folder path. Using an integer or incrementing count will result in duplicate keys for each file (inputs.yml, outputs.yml) in dest

	EnvVarMap[key] = EnvVars{tmp_oo_org, tmp_oo_platform, tmp_oo_assembly, tmp_oo_env}
	return nil
}

func ExtractEnvVars(target_type, ansible_playbook_path *string, logfile *os.File) map[string]EnvVars {

	log.SetOutput(logfile)

	//we will extract env variables from the path of the files itself. we will use filewalk.

	//add a validator to check root_dir variable (the path) if files exists underneath else return here. If path doesn't exist then we will get a nil pointer exception.

	root_dir := (*ansible_playbook_path) + "/vars/conf/" + (*target_type)

	err := filepath.Walk(root_dir, PopulateHash)

	if err != nil {
		log.Fatal("ERROR: Could not map assembly/platform/env together. Check the hierarchy")
	}

	//I have to return map[int]EnvVars because I can't pass a pointer to PopulateHash function. This will have some perf issues.

	return EnvVarMap

}

func Deploy(map_list *map[string]EnvVars, playbook_path, playbook_action, jar_path *string, logfile *os.File, logdir *string, OO_API_TOKEN string) error {
	log.SetOutput(logfile)
	var proccessed chan string

	proccessed = make(chan string)

	//set a time out interval. In case we don't hear from all our go routines within this time, we might be screwed. Ansible output log in /tmp should be helpful to debug why we didn't hear from the workers.
	//For now Deploy is a sync operation..We will wait for results from the goroutines.
	//range over the map and start the go routines.

	for filename, envvariables := range *map_list {

		go RunAnsible(logdir, logfile, playbook_action, playbook_path, jar_path, filename, OO_API_TOKEN, envvariables.OO_ORG, envvariables.OO_ASSEMBLY, envvariables.OO_ENV, envvariables.OO_PLATFORM, proccessed)

	}

	//Collecting the results from all the worker threads.

	for j := 1; j <= len(*map_list); j++ { //can range as well.
		log.Print(<-proccessed)
	}

	return nil

}

//This function will run a multiple GOROUTINES
func RunAnsible(logdir *string, logfile *os.File, playbook_action, playbook_path, jar_path *string, assembly_file, OO_API_TOKEN, OO_ORG, OO_ASSEMBLY, OO_ENV, OO_PLATFORM string, proccessed chan string) {
	log.SetOutput(logfile)
	//45 minute timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3100*time.Second)
	defer cancel()

	playbook_to_run := (*playbook_path) + "/" + "tasks/" + (*playbook_action)

	//we need to modify the OO_PLATFORM as the oneops jar expects -l arguement as platform-<actual platformname>-compute.
	L_OO_PLATFORM := "platform-" + OO_PLATFORM + "-compute"

	cmd := exec.CommandContext(ctx, "ansible-playbook", "-l", L_OO_PLATFORM, "--user=app", "-i", *jar_path, playbook_to_run, "--extra-vars", "OO_ORG="+OO_ORG+" OO_ASSEMBLY="+OO_ASSEMBLY+" OO_PLATFORM="+OO_PLATFORM+" OO_ENV="+OO_ENV+"")
	env := os.Environ()

	env = append(env, fmt.Sprintf("OO_API_TOKEN=%s", OO_API_TOKEN), fmt.Sprintf("OO_ORG=%s", OO_ORG), fmt.Sprintf("OO_ASSEMBLY=%s", OO_ASSEMBLY), fmt.Sprintf("OO_ENV=%s", OO_ENV), fmt.Sprintf("OO_ENDPOINT=%s", "https://oneops.prod.walmart.com/"), fmt.Sprintf("ANSIBLE_HOST_KEY_CHECKING=%s", "False"))
	cmd.Env = env

	//Above note that you can simply add env variables and skip passing --extra-vars in playbook. But to to that you should modify the playbook/template to do lookup('env', 'OO_ENV') instead of checking to the variable directly. Not tested.

	// create a output file.

	outfile, err := os.Create((*logdir) + "/" + assembly_file + ".output")
	if err != nil {
		log.Fatal("Error: Unable to Create output file but I will still proceed")
	}

	defer outfile.Close()

	cmd.Stdout = outfile
	cmd.Stderr = outfile

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		proccessed <- `ERROR: Context deadline exceeded. Ansible is taking more than 50 mins to complete. I Killed ansible process and cleaned all resources. May be network latency or organic growth. if organic growth then consider increasing context timeout for:` + OO_ORG + "_" + OO_ASSEMBLY + "_" + OO_PLATFORM + "_" + OO_ENV
		runtime.Goexit()
	}

	//If there was not context deadline exceeded, then there is some issue in playbook and exited with non 0 exit code.
	if err != nil {
		proccessed <- `WARNING: Playbook failed on some or all hosts with exit code not equal 0. Check ansible output logs for:` + OO_ORG + "_" + OO_ASSEMBLY + "_" + OO_PLATFORM + "_" + OO_ENV
		runtime.Goexit()
	}
	proccessed <- `INFO: Completed deployment with no major failures. Ok to proceed - ` + OO_ORG + "_" + OO_ASSEMBLY + "_" + OO_PLATFORM + "_" + OO_ENV
	runtime.Goexit()

}

func CreateLogFile(log_file_path *string) (*os.File, error) {

	//Make sure the directory path exists, if not create it.

	var logfile *os.File

	if err := os.MkdirAll((*log_file_path), 0744); err != nil {
		return logfile, errors.New("ERROR:Failed to create log file path. Check user permissions")
	}

	//create the actual log file.

	logfile, err := os.OpenFile((*log_file_path)+"/ansible-deployer.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	//we will not close the logfile here. We can close it in the main function. logfile is *os.File (it's an address so it's ok to close in main before exiting)

	if err != nil {
		return logfile, errors.New("ERROR: Can't create log file. I will not start.. sorry")
	}

	return logfile, nil

}
