

# k8sres
Create custom metrics from Kubernetes Resources.

![image](https://github.com/CustomMetrics/k8sres/assets/159939175/b9282376-5b9a-4bd2-b57e-0429844f3c15)

# Kubernetes Resource Metrics Server

This project implements a simple HTTP server that exposes Kubernetes resource metrics via a Prometheus-compatible endpoint. It utilizes Prometheus client libraries along with custom Kubernetes metric collection to facilitate monitoring in Kubernetes environments.

## Features

- **JSON Logging**: Structured logging using Logrus.
- **Environment-Based Debugging**: Debug level logging can be enabled through an environment variable.
- **Prometheus Metrics Endpoint**: Exposes Kubernetes resource metrics at `/metrics`.
- **Kubernetes Configuration**: Dynamically loads Kubernetes configuration for accessing the cluster.

## Getting Started

### Prerequisites

- Go 1.15 or higher
- Access to a Kubernetes cluster
- Prometheus server (for scraping metrics from this service)

### Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/your-repo/kubernetes-metrics-server.git
   cd kubernetes-metrics-server
   ```
2. **Build the application:**
```bash
go build -o metrics-server
```
Set Environment Variables:Optionally, you can enable detailed debug logs by setting the DEBUG environment variable.

```bash
export DEBUG=true
```

3. **Running the Server**
Execute the binary to start the server:

```bash
./metrics-server
The server will start listening on the default port 9805. You can access the metrics at http://localhost:9805/metrics.
```

4. **Configuration**
The service uses the K8sResourceMetricYaml file for setting up resource metric collectors. Ensure this file is correctly configured with the metrics you want to collect.

5. **Deployment**

To deploy this service in a Kubernetes cluster, you can use the provided Dockerfile to containerize the application and then deploy it using Kubernetes resources like Deployment, Service, and potentially an Ingress.

6. **Contributing**

Contributions are welcome. Please open an issue to discuss proposed changes or open a pull request with your updates.


