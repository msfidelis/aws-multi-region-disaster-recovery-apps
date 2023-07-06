import http from 'k6/http';

const base_path = "https://api.msfidelis.com.br"

// const base_path = "http://0.0.0.0:8080"

export default function () {
    let data = {product: "teste", amount: 223.34 };

    let url = `${base_path}/sales`

    let res = http.post(url, JSON.stringify(data), {
        headers: { 'Content-Type': 'application/json' },
      });

      console.log(res.body)
      // console.log(res.json().id); 
}