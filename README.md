### Attach into JVM using Golang and Dynamic Attach API

The demo shows how to attach to running JVM process by using [Dynamic Attach API](https://openjdk.java.net/groups/hotspot/docs/Serviceability.html#battach) and Golang. 

#### How it works

tl;dr:
* check user permissions
* create "attach file"
* send SIGQUIT to the target JVM
* send the command to the socket

I suggest going trough [dunamic_attach.go](dynamic_attach.go)#executeCommand method to see what's inside.


See also blogpost describing what one could do using techniques presented here.