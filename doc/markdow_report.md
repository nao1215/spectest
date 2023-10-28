## GET /image
![Badge](https://img.shields.io/badge/200-green)
  
```mermaid
sequenceDiagram
    autonumber
    cli->>sut: GET /image
    sut-->>cli: 200
```
  
## Event log
#### Event 1
  
GET /image HTTP/1.1  
Host: sut  
  

  
---
  
#### Event 2
  
HTTP/1.1 200 OK  
Content-Length: 40682  
Content-Type: image/png  
  

  
![markdow_report_1.png](markdow_report_1.png)
  
---
  