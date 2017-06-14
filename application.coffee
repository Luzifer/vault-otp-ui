currentTimeout = 0
clipboard = undefined

$ ->
  if $('body').hasClass 'state-signedin'
    initializeApplication()

createOTPItem = (item) ->
  tpl = $('#otp-item').html()

  otpItem = $(tpl)
  otpItem.find('.badge').text item.code.replace(/^(.{3})(.{3})$/, '$1 $2')
  otpItem.find('.title').text item.name
  otpItem.find('i.fa').addClass "fa-#{item.icon}"

  otpItem.appendTo $('#keylist')

delay = (delayMSecs, fkt) ->
  window.setTimeout fkt, delayMSecs

fetchCodes = () ->
  $.ajax
    url: 'codes.json',
    success: updateCodes,
    dataType: 'json',

filterChange = () ->
  filter = $('#filter').val().toLowerCase()
  $('.otp-item').each (idx, el) ->
    if $(el).find('.title').text().toLowerCase().match(filter) == null
      $(el).hide()
    else
      $(el).show()

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

tick = (delay, fkt) ->
  window.setInterval fkt, delay

timeLeft = () ->
  now = new Date().getTime()
  (currentTimeout - now) / 1000

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
