# local_share
Golang Program to share files via Local Intranet ( LAN )


#### Run the below command in sender

```bash
go run main.go send

Enter folder path: /path/to/share
Enter receiver IP: 192.168.0.11
Enter receiver port (default 8080): 8080
```

#### Run the below command in receiver


```bash
go run main.go receive
```