./build.sh
docker login
docker tag atlantis/summary:latest  dragonliu1995/page_summary:v2
docker push dragonliu1995/page_summary:v2
ssh -i C:/Users/LXL/Desktop/info441/info441.pem ec2-user@ec2-3-229-233-121.compute-1.amazonaws.com << EOF
    docker rm -f pageSummary_container
    docker pull dragonliu1995/page_summary:v2
    docker run -d -p 443:4000 -v /etc/letsencrypt:/tmp/cert:ro -e ADDR=:4000  -e TLSCERT=/tmp/cert/live/api.atlantisking.me/fullchain.pem -e TLSKEY=/tmp/cert/live/api.atlantisking.me/privkey.pem --name pageSummary_container dragonliu1995/page_summary:v2
EOF