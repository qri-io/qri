# Troubleshooting

This is a list of commonly encountered problems, known issues, and their solutions.

### The following error occurs with `qri connect` :
 `ERROR mdns: mdns lookup error: failed to bind to any unicast udp port mdns.go:140 ...`

This is caused by a limit on the number of files allowed when qri is trying to connect to the distributed web. You can increase the open file descriptor limit by entering the following command `ulimit -n 2560` before running `qri connect`. See [https://github.com/ipfs/support/issues/17](https://github.com/ipfs/support/issues/17) for details.


### The following error occurs with `qri command not working` :
 `getting the qri binary on your $PATH`
    
This is caused by $PATH not containing a reference to $GOPATH/bin. To alleviate this problem try:
```bash
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```
See [https://stackoverflow.com/questions/21001387/how-do-i-set-the-gopath-environment-variable-on-ubuntu-what-file-must-i-edit/21012349#21012349](https://stackoverflow.com/questions/21001387/how-do-i-set-the-gopath-environment-variable-on-ubuntu-what-file-must-i-edit/21012349#21012349) for details. 
