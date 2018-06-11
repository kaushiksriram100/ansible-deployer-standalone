# ansible-deployer-tp

A standalone solution to run a ansible playbook in several oneops environments.

0. Please read - https://confluence.walmart.com/x/0ZybD
1. Install ansible software in you machine. 
2. Compile this code
3. clone the ansible playbook
 -- Steps 1-3 can be done using a simple playbook so that the environment is under a proper config management. 

4. Run deployer like this - 

5. To compile this code for oneops Machines. Copy the resulting binary to the files folder in this playbook -> https://gecgithub01.walmart.com/pulse/wm-tp-deployer-setup/tree/master/files
   ``` env GOOS=linux GOARCH=amd64 go build -v shyun.go ```

```
/opt/tp-deployer/shyun -playbookpath /opt/tp-deployer/ansible-stream-splitter/ -playbookaction site.yml -targettype "inventories/pulse/CDC/prod-BM/" -invjar /opt/tp-deployer/oneops-inventory/oo-wrapper.py --tags "stream-splitter" --mfp 50 --s1 5 --s2 10
```

Dependency: 

1. Clone this oneops jar to get the inventory list from oneops - https://github.com/oneops/oneops-inventory to your deployment VM.
2. Ansible deployment VMs are able to connect to target VMs with specified user. 
3. Keys are properly deployed. 

add these keys:
```
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3aP+4Bfmwcj7Zz7M6szJSInWGkKsQHb84Evtz2groOvlAsszZI9v3TxH6sTz9JCOi766a8PQszuy+SxdSiSWPPYzFP8rap+bAjvHcmEfbykX4D0F1dke0dXjkW7/hzm6Lej/iexNId5cn8vx0IfISmPmSdfisVEBJ1cI6OblXQixmDz6ogQmvPLGqd39yQ5U0zhNHPTh+EnBYItMJLdmQNnjwxAoj4Qx11+NJdFFp8JQmbRQsNNETtscUR0ZeSrvylOb4tSzqVRsH34OwLGTluTN0q03rdjo9Qsaae+JD1dDWi34heuwzYBW3Y45KvAwtF7/DS4e1KrugUaD9+fzb tpdeployers
```


Author: Sriram Kaushik

Note: Open for feature enhancements, suggestions and PRs and bug-fixes. 
