# ansible-deployer-standalone
A standalone solution to run a ansible playbook in several oneops environments.

1. Install ansible software in you machine.
2. Compile this code
3. clone the ansible playbook
4. Run the code like this

```
shyun_app_user -targettype <githubpath> -playbookpath /opt/splunk-deployer/wm-splunk-universal-forwarder/ -playbookaction main.yml -invjar /opt/splunk-deployer/oneops-inventory/oo-wrapper.py --logfile /var/tmp/deploy-splunk-uf_app_user/log/ > /dev/null 2>&1
```


Author: Sriram Kaushik
