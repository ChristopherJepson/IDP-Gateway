# 1. THE BASE IMAGE
# We start with "Alpine" Linux. It is an ultra-secure, stripped-down version of Linux 
# that is only 5MB in size. We use the version that already has Go installed.
FROM golang:alpine

# 2. INSTALL DEPENDENCIES
# Alpine uses 'apk' instead of 'apt' to install software. 
# We tell it to download and install FFmpeg, skipping the local cache to keep the container tiny.
RUN apk add --no-cache ffmpeg

# 3. SET THE WORKSPACE
# We create a folder called /app inside the container and make it our active directory.
WORKDIR /app

# 4. COPY THE CODE
# We copy our Go module file and our main.go file from your Windows machine 
# into the /app folder inside the Linux container.
COPY go.mod ./
COPY main.go ./

# 5. COMPILE THE APP
# We tell Go to compile our raw code into a highly optimized, standalone executable 
# file named "api-gateway".
RUN go build -o api-gateway main.go

# 6. OPEN THE FIREWALL
# We document that this container communicates on port 8080.
EXPOSE 8080

# 7. THE STARTUP COMMAND
# When Portainer eventually starts this container, this is the exact command it runs 
# to turn the server on.
CMD ["./api-gateway"]