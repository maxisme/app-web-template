To implement `appserver` create a new project:

1. Create an `images/` directory which contains:
    - icon.ico
    - logo.png
    - notifications.png
    - og_logo.png
    
    Use [logoToImages.sh](https://github.com/maxisme/App-Deployment-Tools/blob/master/Media/logoToImages.sh)
2. Create a `templates/` directory which contains the pages you want the website template to contain (can be written in go templates also the `.Data` variable contains any GET http arguments passed)

3. Add your latest `.dmg` file to the root.

4. Create a `main.go` with this content: 
    ```go
    package main
    
    import (
        "github.com/maxisme/appserver"
    )
    
    func main() {
        conf := appserver.ProjectConfig{
            Name: "",
            Host: "",
            DmgPath: "",
            KeyWords: "",
            Description: "",
            Recaptcha: appserver.Recaptcha{
                Pub: "",
                Priv: "",
            },
            Sparkle: appserver.Sparkle{
                Version:1,
                Description:"",
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

6. Create an `appserver.service` file (customising `Description` and also `WorkingDirectory` with where the root of this project is) and place in the `/etc/systemd/system` directory:
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
   
7. Create an `appserver.socket` file and place in the `/etc/systemd/system` directory:
    ```bash
    [Socket]
    ListenStream = 8080
    
    [Install]
    WantedBy=sockets.target
    ```

8. Start the service by running:
    ```bash
    $ systemctl daemon-reload
    $ systemctl start appserver.socket
    ```
    
9. Now test the service is running by executing:
    ```bash
    $ curl 127.0.0.1:8080
    ```
    

Every time you update simply rerun step `4` and then `$ systemctl restart appserver.service`

Use `$ journalctl -u appserver.service` to debug any issues you may have.
