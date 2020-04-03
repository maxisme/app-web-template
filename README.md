To implement `appserver` create a new project:

1. Create an `images/` directory which contains:
    - icon.ico
    - logo.png
    - notifications.png
    - og_logo.png
    
    Use [logoToImages.sh](https://github.com/maxisme/App-Deployment-Tools/blob/master/Media/logoToImages.sh)
2. Create a `templates/` directory which contains the pages you want the website template to contain (can be written in go templates also the `.Data` variable contains any GET http arguments passed)

3. Add your latest `.dmg` file to the root.

4. Create a `main.go` file: 
    ```go
    package main
    
    import (
        "github.com/maxisme/appserver"
	    "os"
    )
    
    func main() {
        conf := appserver.ProjectConfig{
            Name: "",
            Host: "",
            DmgPath: "",
            KeyWords: "",
            Description: "",
            Recaptcha: appserver.Recaptcha{
                Pub: os.Getenv("captch-pub"),
                Priv: os.Getenv("captch-priv"),
            },
            Sparkle: appserver.Sparkle{
                Version: "0.1",
                Description: "",
            },
        }
    
        if err := appserver.Serve(conf); err != nil {
            panic(err)
        }
    }
    
    ```

5. Build binary for project:
    ```bash
    $ go build -o /usr/local/bin/appserver main.go
    ```

6. Create `/etc/systemd/system/appserver.service` file, customising:
    - `Description` 
    - `WorkingDirectory` with the root path of your project
    - Also add any environment variables to your projects `.env`
    
   ```bash
   [Unit]
   Description=
   Requires=network.target
   After=multi-user.target
   
   [Service]
   Type=simple
   WorkingDirectory=
   ExecStart=/usr/local/bin/appserver
   ExecReload=/bin/kill -SIGINT $MAINPID
   
   [Install]
   WantedBy=multi-user.target
   ```
   
7. Create an `/etc/systemd/system/appserver.socket` file:
    ```bash
    [Socket]
    ListenStream = 8080
    
    [Install]
    WantedBy=sockets.target
    ```

8. Start the service by running:
    ```bash
    $ systemctl daemon-reload
    $ systemctl enable appserver.socket
    $ systemctl start appserver.socket
    $ systemctl status appserver.socket
    ```
    
9. Now test the service is running by executing:
    ```bash
    $ curl 127.0.0.1:8080
    ```
    

A deploy script may look like:
```bash
# pull latest from project you have created
git fetch origin
git checkout master
git pull

# get latest version from this project
go get -u github.com/maxisme/appserver

# build binary
go build -o /usr/local/bin/appserver main.go

# reload binary with 0 downtime
systemctl restart appserver.service
```

Use `$ journalctl -u appserver.service` to debug any issues you may have.
