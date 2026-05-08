FROM public.ecr.aws/lambda/provided:al2023 AS build

RUN dnf install -y golang && dnf clean all

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap ./main.go

FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /src/bootstrap /var/runtime/bootstrap

ENTRYPOINT ["/var/runtime/bootstrap"]
