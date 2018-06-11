/* PURPOSE: To create a worker which can run the ansible-playbook command for each env/host in oneops.
AUTHOR: SRIRAM KAUSHIK
*/

package deployer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Inputs struct {
	Log_file_path           *string
	Ansible_playbook_path   *string
	Ansible_playbook_action *string
	Oneops_jar_path         *string
	Ansible_tags            *string
	Ansbile_skip_tags       *string
	Target_type             *string
	Ansible_user            *string
	Hosts_limit             *string
	Max_fail_percentage     *string
	S1                      *string
	S2                      *string
	S3                      *string
}

type EnvVars struct {
	OO_ORG          string
	OO_PLATFORM     string
	OO_ASSEMBLY     string
	OO_ENV          string
	TP_PROJECT      string
	TP_DC           string
	TP_ENV          string
	IS_HOSTS_INI    bool
	HOSTS_INI_LIMIT string
}

var EnvVarMap = make(map[string]EnvVars)

func ExtractOneOpsVars(propfile string) (oo_org, oo_env, oo_platform, oo_assembly string, err error) {

	//create some regex variables

	var MatchOOOrg = regexp.MustCompile("^OO_ORG=.+")
	var MatchOOAssembly = regexp.MustCompile("^OO_ASSEMBLY=.+")
	var MatchOOPlatform = regexp.MustCompile("^OO_PLATFORM=.+")
	var MatchOOEnv = regexp.MustCompile("^OO_ENV=.+")

	//scan the deployment properties file to get all the oneops related configuration
	filehandle, err := os.Open(propfile)

	if err != nil {
		return
	}

	defer filehandle.Close()

	filescanner := bufio.NewScanner(filehandle)

	for filescanner.Scan() {
		switch true {
		case MatchOOOrg.MatchString(filescanner.Text()):
			oo_org = strings.Split(filescanner.Text(), "=")[1]
		case MatchOOEnv.MatchString(filescanner.Text()):
			oo_env = strings.Split(filescanner.Text(), "=")[1]
		case MatchOOAssembly.MatchString(filescanner.Text()):
			oo_assembly = strings.Split(filescanner.Text(), "=")[1]
		case MatchOOPlatform.MatchString(filescanner.Text()):
			oo_platform = strings.Split(filescanner.Text(), "=")[1]
		}
	}

	//return will return all the variables for this function
	return
}

func PopulateHash(path string, fp os.FileInfo, err error) error {
	if fp.IsDir() {
		return nil
	}

	if path == "" {
		return nil
	}

	switch filename := fp.Name(); filename {

	case "deployment.properties":

		tmp := strings.Split(path, "/")
		tmp_tp_env := tmp[len(tmp)-2]
		tmp_tp_dc := tmp[len(tmp)-3]
		tmp_tp_project := tmp[len(tmp)-4]

		tmp_oo_org, tmp_oo_env, tmp_oo_platform, tmp_oo_assembly, err := ExtractOneOpsVars(path)

		if err != nil {
			return err
		}
		key := tmp_oo_org + "_" + tmp_oo_assembly + "_" + tmp_oo_platform + "_" + tmp_oo_env + "_" + tmp_tp_env + "_" + tmp_tp_dc + "_" + tmp_tp_project

		//This hash generates a key which is the folder path. Using an integer or incrementing count will result in duplicate keys for each file (inputs.yml, outputs.yml) in dest
		//********* add a check if any key element in empty don't process it. *********

		EnvVarMap[key] = EnvVars{OO_ORG: tmp_oo_org, OO_PLATFORM: tmp_oo_platform, OO_ASSEMBLY: tmp_oo_assembly, OO_ENV: tmp_oo_env, TP_PROJECT: tmp_tp_project, TP_DC: tmp_tp_dc, TP_ENV: tmp_tp_env}
		return nil

	case "hosts.ini":

		tmp := strings.Split(path, "/")
		tmp_tp_env := tmp[len(tmp)-2]
		tmp_tp_dc := tmp[len(tmp)-3]
		tmp_tp_project := tmp[len(tmp)-4]

		key := "hostsini_" + tmp_tp_env + "_" + tmp_tp_dc + "_" + tmp_tp_project
		EnvVarMap[key] = EnvVars{TP_PROJECT: tmp_tp_project, TP_DC: tmp_tp_dc, TP_ENV: tmp_tp_env, IS_HOSTS_INI: true}
		return nil
	default:
		return nil

	}

}

func ExtractEnvVars(inputs *Inputs, logfile *os.File) map[string]EnvVars {

	log.SetOutput(logfile)

	//we will extract env variables from the path of the files itself. we will use filewalk.

	//add a validator to check root_dir variable (the path) if files exists underneath else return here. If path doesn't exist then we will get a nil pointer exception.

	root_dir := *inputs.Ansible_playbook_path + "/" + *inputs.Target_type //can do just inputs.Target_type. But this is just to better readability

	err := filepath.Walk(root_dir, PopulateHash)

	if err != nil {
		log.Fatal("ERROR: Could not map assembly/platform/env together. Check the hierarchy")
	}

	//maps are passed as references.

	return EnvVarMap

}

func Deploy(map_list *map[string]EnvVars, inputs *Inputs, logfile *os.File, OO_API_TOKEN string) error {
	log.SetOutput(logfile)
	var proccessed chan string

	proccessed = make(chan string)

	//set a time out interval. In case we don't hear from all our go routines within this time, we might be screwed. Ansible output log in /tmp should be helpful to debug why we didn't hear from the workers.
	//For now Deploy is a sync operation..We will wait for results from the goroutines.
	//range over the map and start the go routines.

	for filename, envvariables := range *map_list {

		go RunAnsible(inputs, logfile, filename, OO_API_TOKEN, envvariables.OO_ORG, envvariables.OO_ASSEMBLY, envvariables.OO_ENV, envvariables.OO_PLATFORM, envvariables.TP_PROJECT, envvariables.TP_DC, envvariables.TP_ENV, envvariables.IS_HOSTS_INI, proccessed)

	}

	//Collecting the results from all the worker threads.

	for j := 1; j <= len(*map_list); j++ { //can range as well.
		log.Print(<-proccessed)
	}

	return nil

}

//This function will run a multiple GOROUTINES
func RunAnsible(inputs *Inputs, logfile *os.File, assembly_file, OO_API_TOKEN, OO_ORG, OO_ASSEMBLY, OO_ENV, OO_PLATFORM, TP_PROJECT, TP_DC, TP_ENV string, is_hosts_ini bool, proccessed chan string) {
	log.SetOutput(logfile)
	//45 minute timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3100*time.Second)
	defer cancel()

	playbook_to_run := *inputs.Ansible_playbook_path + "/" + *inputs.Ansible_playbook_action

	//we need to modify the OO_PLATFORM as the oneops jar expects -l arguement as platform-<actual platformname>-compute.
	var L_OO_PLATFORM string

	if !is_hosts_ini {
		L_OO_PLATFORM = "platform-" + OO_PLATFORM + "-compute"
	} else {
		L_OO_PLATFORM = *inputs.Hosts_limit
	}

	cmd := exec.CommandContext(ctx, "ansible-playbook", "-l", L_OO_PLATFORM, "-u", *inputs.Ansible_user, "-i", *inputs.Oneops_jar_path, playbook_to_run, "--tags", *inputs.Ansible_tags, "--skip-tags", *inputs.Ansbile_skip_tags, "--extra-vars", "project="+TP_PROJECT+" data_center="+TP_DC+" env="+TP_ENV+" ans_max_fail_percent="+*inputs.Max_fail_percentage+" s1="+*inputs.S1+" s2="+*inputs.S2+" s3="+*inputs.S3+"")
	env := os.Environ()

	env = append(env, fmt.Sprintf("OO_API_TOKEN=%s", OO_API_TOKEN), fmt.Sprintf("OO_ORG=%s", OO_ORG), fmt.Sprintf("OO_ASSEMBLY=%s", OO_ASSEMBLY), fmt.Sprintf("OO_ENV=%s", OO_ENV), fmt.Sprintf("OO_ENDPOINT=%s", "https://oneops.prod.walmart.com/"), fmt.Sprintf("ANSIBLE_HOST_KEY_CHECKING=%s", "False"))
	cmd.Env = env

	//Above note that you can simply add env variables and skip passing --extra-vars in playbook. But to to that you should modify the playbook/template to do lookup('env', 'OO_ENV') instead of checking to the variable directly. Not tested.

	// create a output file.

	outfile, err := os.Create((*inputs.Log_file_path) + "/" + assembly_file + ".output")
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
