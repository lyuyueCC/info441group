GOOS="linux"
# docker pull nginx
# docker run -d --name tmp-nginx nginx
# docker cp tmp-nginx:/etc/nginx/conf.d/default.conf default.conf
# docker rm -f tmp-nginx
docker build -t dragonliu1995/summary_client .