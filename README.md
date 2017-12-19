# ansible-deployer-standalone

A standalone solution to run a ansible playbook in several oneops environments.

1. Install ansible software in you machine. 
2. Compile this code
3. clone the ansible playbook
 -- Steps 1-3 can be done using a simple playbook so that the environment is under a proper config management. 

4. Run deployer like this - 

```
shyun_app_user -targettype <githubpath> -playbookpath /opt/splunk-deployer/wm-splunk-universal-forwarder/ -playbookaction main.yml -invjar /opt/splunk-deployer/oneops-inventory/oo-wrapper.py --logfile /var/tmp/deploy-splunk-uf_app_user/log/ > /dev/null 2>&1
```

Dependency: 

1. Clone this oneops jar to get the inventory list from oneops - https://github.com/oneops/oneops-inventory to your deployment VM.
2. Ansible deployment VMs are able to connect to target VMs with specified "app" user. 
3. Keys are properly deployed. 

Author: Sriram Kaushik

Note: Open for feature enhancements, suggestions and PRs and bug-fixes. 
