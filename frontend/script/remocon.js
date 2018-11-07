/**
 * remocon.js
 */

riot.mount("signals", {
  api: {
    "status": "ok",
    "signals": [{ "remote": "light", "name": "on" }, { "remote": "light", "name": "off" }, { "remote": "light", "name": "night" }]
  },
  send: send,
  storeEndpoint: storeEndpoint,
});

function storeEndpoint(e) {
  e.preventDefault();
  localStorage.setItem("endpoint", e.target.endpoint.value);
  localStorage.setItem("device_id", e.target.device_id.value);
}

function send(item) {
  console.log(item);
  var data = {
    ...item,
    device_id: localStorage.getItem("device_id"),
  }
  var endpoint = localStorage.getItem("endpoint");
  return fetch(endpoint, {
    method: "POST",
    mode: "cors",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  }).then(res => {
    console.log(res);
    return res.json();
  }).then(data => {
    console.log(data);
  }).catch(err => {
    console.log(err);
    // TODO: 適切なエラーハンドリング
    alert(err);
  });
}
