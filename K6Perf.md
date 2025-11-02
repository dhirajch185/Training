Install K6 on windows using ` winget install k6 --source winget `

Create a .js file and import the k6 libraries. ` import http from 'k6/http'; `

create a function and use http.get call to generate load on the url.

example: 
```
import http from 'k6/http';

export default function () {
   http.get("https://quickpizza.grafana.com/test.k6.io/"); 
}
```

