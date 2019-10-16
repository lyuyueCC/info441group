./build.sh
docker login
docker tag dragonliu1995/summary_client:latest dragonliu1995/summary_client:v2
docker push dragonliu1995/summary_client:v2
ssh -i C:/Users/LXL/Desktop/info441/info441.pem ec2-user@ec2-52-45-82-197.compute-1.amazonaws.com << EOF
    docker rm -f summary_client_container
    docker pull dragonliu1995/summary_client:v2
    docker run -d -p 80:80 -p 443:443 -v /etc/letsencrypt:/tmp/cert:ro -e TLSCERT=/tmp/cert/live/api.atlantisking.me/fullchain.pem -e TLSKEY=/tmp/cert/live/api.atlantisking.me/privkey.pem --name summary_client_container dragonliu1995/summary_client:v2
EOF