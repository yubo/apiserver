## apiserver s3

### S3

<details><summary> install minio </summary>

```
$ brew install minio
$ brew services run minio
```

访问管理页面

```yaml
url: http://localhost:9000
user: minioadmin
pass: minioadmin
```

创建 一个名字叫 test 的 bucket

```
buckets -> create bucket
```

修改 bucket 策略

```
buckets -> test -> Manager -> summay -> access Policy -> public
```

提交后，修改 public -> custom, 只允许匿名 get object

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:GetObject"
            ],
            "Resource": [
                "arn:aws:s3:::test/*"
            ]
        }
    ]
}
```

</details>

### server

```sh
$ go run ./main.go -f ./config.yaml
```

### client

```sh
$ echo 123 > /tmp/test.txt
$ curl -X POST -F "uploadFile=@/tmp/test.txt" http://localhost:8080/s3/tmp
"tmp/test.txt"
```

get file
```sh
$ curl -L  -X GET  http://localhost:8080/s3/tmp/test.txt
123
```

delete file
```sh
$ curl -X DELETE http://localhost:8080/s3/tmp/test.txt
```


