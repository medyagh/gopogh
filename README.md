# goprettyorgohome
### Convert golang tests output to human friendly html.

This tool is built based on this minikube PR by @tstromberg https://github.com/kubernetes/minikube/pull/5225 it is used by minikube for prettifying tests outputs.

Example report:  
- [raw output](https://storage.googleapis.com/minikube-builds/logs/6258/VirtualBox_Linux.txt) (before gopogh)
- [html output](https://storage.googleapis.com/minikube-builds/logs/6258/VirtualBox_Linux.html) (after gopogh)

### Install linux
```
        curl -LO https://github.com/medyagh/gopogh/releases/download/v0.1.15/gopogh-linux-amd64
        sudo install gopogh-linux-amd64 /usr/local/bin/gopogh
```

### Install macos
```
        curl -LO https://github.com/medyagh/gopogh/releases/download/v0.1.15/gopogh-linux-amd64
        sudo install gopogh-linux-amd64 /usr/local/bin/gopogh
```

### Install windows
download latest binary from :
https://github.com/medyagh/gopogh/releases


### Example usage in github actions

see minikube's workflow 
https://github.com/kubernetes/minikube/blob/master/.github/workflows/main.yml


#### features:
- separate tests by results.
- make test results foldable.
- search in each test result separately.
- table of content with each test duration.


### Before
<img src="https://raw.githubusercontent.com/medyagh/gopogh/master/images/before.png" width="738" alt="before">

### After
<img src="https://raw.githubusercontent.com/medyagh/gopogh/master/images/after1.png" width="738" alt="after1">

### After
<img src="https://raw.githubusercontent.com/medyagh/gopogh/master/images/after2.png" width="738" alt="after2">


### Try localy 
```
make test
open output.html
```