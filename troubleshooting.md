# Troubleshooting

This is a list of commonly encountered problems, known issues, and their solutions.

### The following error occurs with `qri connect`: `ERROR mdns: mdns lookup error: failed to bind to any unicast udp port mdns.go:140 ...`

This is caused by a limit on the number of files allowed when qri is trying to connect to the distributed web. You can increase the open file descriptor limit by entering the following command `ulimit -n 2560` before running `qri connect`. See [https://github.com/ipfs/support/issues/17](https://github.com/ipfs/support/issues/17) for details.



	
