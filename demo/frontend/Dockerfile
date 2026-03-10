FROM nginx:latest
#RUN apt-get update && apt-get install -y iputils-ping
RUN apt-get update -y
RUN apt install -y iputils-ping
WORKDIR /app
COPY ./frontend/index.html /usr/share/nginx/html/
COPY ./frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
