## POST /hello
![Badge](https://img.shields.io/badge/200-green)
  
```mermaid
sequenceDiagram
    autonumber
    cli->>sut: POST /hello
    sut-->>cli: 200
```
  
## Event log
#### Event 1
  
POST /hello HTTP/1.1  
Host: sut  
  

  
---
  
#### Event 2
  
HTTP/1.1 200 OK  
Connection: close  
Content-Type: application/json  
  

  
```json
{
  "a": 12345
}
```
  
---
  