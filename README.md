# PAID CASHU BLOSSOM SERVER

This is a blossom server that charger for usage of it API for with cashu. Upload an download.

# how to run the blossom server?

## Set the ENVIROMENT VARIABLES

***Mandatory ENV Variables:***
field: DOMAIN, TRUSTED_MINT, SEED.

If you don't set DOWNLOAD_COST_2MB and UPLOAD_COST_2MB they will be set to 0. 

## configure you caddy server.
Caddy is used for reverse proxy and tls handling and creation. Please change the following fields to your correct
values:

example.com: should be the same domain as the DOMAIN env variable.
your-email@domain.com: to a domain that you control. This is for lets encrypt notifiations.


## run the server.
After you set the .env file or env variables. you can run the blossom server by running the command:

```
go run ./...
```



### TODOs

1. Make way do you can use certs directly in the server.  
2. Send tokens to a npub via direct message. (Blocked by the go-nostr version I'm using)


