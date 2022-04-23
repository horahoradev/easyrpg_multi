# Ynoclient Dockerfile
1. ./install.sh
2. sudo docker build .
3. sudo docker run -p 80:80 <image from step 2>
4. visit localhost in browser

This setup does not include the server. By default, the client assumes the server is available at localhost:8080.

Only modify ynoclient_modified , otherwise caching won't work.
