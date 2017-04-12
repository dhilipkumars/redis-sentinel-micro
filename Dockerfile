FROM busybox 
ADD ./redis_sentinel_k8s /
ENTRYPOINT ["/redis_sentinel_k8s"]
