currentTimeout = 0
clipboard = undefined

# document-ready function to start Javascript processing
$ ->
  if $('body').hasClass 'state-signedin'
    initializeApplication()

# createOTPItem generates code entries from the JSON objects passed from the backend
createOTPItem = (item) ->
  tpl = $('#tpl-otp-item').html()

  otpItem = $(tpl)
  otpItem.find('.badge').text item.code.replace(/^(.{3})(.{3})$/, '$1 $2')
  otpItem.find('.title').text item.name
  otpItem.find('i.fa').addClass "fa-#{item.icon}"

  otpItem.appendTo $('#keylist')

# createAlert adds a colored message at the top of the list
# type = success / info / warning / danger
createAlert = (type, keyword, message, timeout) ->
  tpl = $('#tpl-message').html()

  alrt = $(tpl)
  alrt.addClass "alert-#{type}"
  alrt.find('.keyword').text keyword
  alrt.find('.message').text message

  alrt.appendTo $('#messagecontainer')

  if timeout > 0
    delay timeout, () ->
      alrt.remove()

# delay is a convenience wrapper to swap parameters of setTimeout
delay = (delayMSecs, fkt) ->
  window.setTimeout fkt, delayMSecs

# fetchCodes contacts the backend to receive JSON containing current codes
fetchCodes = () ->
  $.ajax
    url: 'codes.json',
    success: updateCodes,
    dataType: 'json',
    error: () ->
      createAlert 'danger', 'Oops.', 'Server could not be contacted. Maybe you (or the server) are offline? I will retry in a few seconds.', 5000
      delay 5000, fetchCodes
    statusCode:
      401: () ->
        window.location.reload()
      500: () ->
        createAlert 'danger', 'Oops.', 'The server responded with an internal error. I will retry in a few seconds.', 2000
        delay 2000, fetchCodes

# filterChange is called when changing the filter field and matches the
# titles of all shown entries. Those not matching the given regular expression
# will be hidden. The filterChange function is also called after a successful
# refresh of the shown codes to re-apply
filterChange = () ->
  filter = $('#filter').val().toLowerCase()
  $('.otp-item').each (idx, el) ->
    if $(el).find('.title').text().toLowerCase().match(filter) == null
      $(el).hide()
    else
      $(el).show()

# initializeApplication initializes some basic events and starts the first
# polling for codes
initializeApplication = () ->
  $('#keylist').empty()
  $('#filter').bind 'keyup', filterChange
  tick 500, refreshTimerProgress
  fetchCodes()

# refreshTimerProgress updates the top progressbar to display the
# remaining time until the one-time-passwords changes
refreshTimerProgress = () ->
  secondsLeft = timeLeft()
  $('#timer').css 'width', "#{secondsLeft / 30 * 100}%"

# tick is a convenience wrapper to swap parameters of setInterval
tick = (delay, fkt) ->
  window.setInterval fkt, delay

# timeLeft calculates the remaining time until codes get invalid
timeLeft = () ->
  now = new Date().getTime()
  (currentTimeout - now) / 1000

# updateCodes is being called when the backend delivered codes. The codes
# are then rendered and the clipboard methods are re-bound. Afterwards the
# next fetchCodes call is timed to that moment when the codes are getting
# invalid
updateCodes = (data) ->
  currentTimeout = new Date(data.next_wrap).getTime()

  if clipboard
    clipboard.destroy()

  $('#keylist').empty()
  for token in data.tokens
    createOTPItem token

  clipboard = new Clipboard '.otp-item',
    text: (trigger) ->
      $(trigger).find('.badge').text().replace(' ', '')

  filterChange()

  delay timeLeft()*1000, fetchCodes
