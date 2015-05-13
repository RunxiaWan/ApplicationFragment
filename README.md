These codes try to implement proxys for Application Fragement. Application Fragement fragements Large package in DNS level to solve package size problem in DNS.

The client proxy is the proxy which supposed to be run in client side. It will listen on port 53 and redirect DNS traffic to Application Fragement server through a specific port.

The server proxy is the proxy which supposed to be run in Server side. It will listen on the specific port, and resolve the query.