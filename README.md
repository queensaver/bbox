# bbox

bbox is the server component to the bhive

## install

```bash
curl https://raw.githubusercontent.com/queensaver/bbox/master/ansible/install.sh | bash
```

## info

Authenticate against the server by getting a token and saving it in a file called `~/.queensaver_token` (just the token, without the linefeed): 

```bash
curl --cookie-jar - -X POST -F 'username=your-username' -F 'password=your-password' https://api.queensaver.com/v1/login
```

Install the bbox software with the following command on your raspberry pi: 

```bash
curl https://raw.githubusercontent.com/queensaver/bbox/master/ansible/install.sh | bash
```

To send a varroa image without the bhive: 

```bash
curl -v -F bhiveId=b827ebe6b396 -F epoch=1649171777 -F scan=@varroa.jpg http://localhost:8333/varroa
```

To send a fake temperature to a local bhive
```bash
curl -X POST -d '{"temperature": 22.24, "BBoxID": "aa:bb:cc:dd:ee:ff"}' http://localhost:8333/temperature
```
