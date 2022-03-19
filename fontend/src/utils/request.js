export const post = (url, data, headers) => {
  headers = Object.assign({
    'Content-Type': 'application/json'
  }, headers)
  return fetch('api/' + url, {
    method: 'POST',
    body: JSON.stringify(data),
    headers: new Headers(headers)
  })
}