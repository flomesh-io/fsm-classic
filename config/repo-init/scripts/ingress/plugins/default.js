pipy()

.pipeline('request')
  .replaceMessage(
    new Message({ status: 503 }, 'Service Unavailable')
  )
