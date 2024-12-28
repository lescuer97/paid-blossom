# PAID CASHU BLOSSOM SERVER

This is a blossom server that charger for usage of it API for with cashu. Upload an download.

# how to run the blossom server?

## Set the ENVIROMENT VARIABLES

***Mandatory ENV Variables:***
field: DOMAIN, TRUSTED_MINT, SEED.

If you don't set DOWNLOAD_COST_2MB and UPLOAD_COST_2MB they will be set to 0. 

## configure you caddy file (if want to use reverse proxy).
Caddy is used for reverse proxy and tls handling and creation. Please change the following fields to your correct
values:

example.com: should be the same domain as the DOMAIN env variable.
your-email@domain.com: to a domain that you control. This is for lets encrypt notifiations.


## run the server.
Right now running this servers is still not very straight forward. You can run it directly on the command line like
this. You will need to set the env variables as said before. 

```
go build -o ratasker ./cmd/ratasker/main.go && ./ratasker
```

## The way I run it (as a service). 

I run the paid blossom as a service in my Linux box. I use two files in the repo to configure this. Caddyfile and
ratasker.service. 

**If you are going to use caddy as reverse proxy you will need to point your DNS to your box so you get
a valid TLS Cert.**

### Steps:
1. Get ratasker binary to the /usr/bin folder.
2. Change ratasker.service file  to use your correct name and enviroment variables or point or point to .env file.
   Get it into the `/etc/systemd/system/` directory
3. Change modify /etc/caddy/Caddyfile file to use the custom one on the repo.
4. Enable and run the caddy service and the ratasker.service
    ```
    sudo systemctl daemon-reload
    sudo systemctl enable ratasker
    sudo systemctl start ratasker
    sudo systemctl enable caddy
    sudo systemctl start caddy
    ```



### TODOs

1. Improve code quality.
2. Make installable package for Linux.
2. Send tokens to a npub via direct message. (Blocked by the go-nostr version I'm using)


