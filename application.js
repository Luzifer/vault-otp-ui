// Generated by CoffeeScript 1.12.4
(function() {
  var clipboard, createAlert, createOTPItem, currentTimeout, delay, fetchCodes, fetchInProgress, filterChange, initializeApplication, iterationCurrent, iterationNext, preFetch, refreshTimerProgress, serverConnectionError, tick, timeLeft, updateCodes, updatePreFetch;

  currentTimeout = 0;

  clipboard = null;

  preFetch = null;

  fetchInProgress = false;

  serverConnectionError = false;

  iterationCurrent = 'current';

  iterationNext = 'next';

  $(function() {
    if ($('body').hasClass('state-signedin')) {
      return initializeApplication();
    }
  });

  createOTPItem = function(item) {
    var otpItem, tpl;
    tpl = $('#tpl-otp-item').html();
    otpItem = $(tpl);
    otpItem.find('.badge').text(item.code.replace(/^(.{3})(.{3})$/, '$1 $2'));
    otpItem.find('.title').text(item.name);
    otpItem.find('i.fa').addClass("fa-" + item.icon);
    return otpItem.appendTo($('#keylist'));
  };

  createAlert = function(type, keyword, message, timeout) {
    var alrt, tpl;
    tpl = $('#tpl-message').html();
    alrt = $(tpl);
    alrt.find('.alert').addClass("alert-" + type);
    alrt.find('.alert').find('.keyword').text(keyword);
    alrt.find('.alert').find('.message').text(message);
    alrt.appendTo($('#messagecontainer'));
    if (timeout > 0) {
      return delay(timeout, function() {
        return alrt.remove();
      });
    }
  };

  delay = function(delayMSecs, fkt) {
    return window.setTimeout(fkt, delayMSecs);
  };

  fetchCodes = function(iteration) {
    var data, successFunc;
    if (fetchInProgress) {
      return;
    }
    fetchInProgress = true;
    if (iteration === iterationCurrent) {
      successFunc = updateCodes;
    } else {
      successFunc = updatePreFetch;
    }
    if (iteration === iterationCurrent && preFetch !== null) {
      data = preFetch;
      preFetch = null;
      successFunc(data);
      return;
    }
    return $.ajax({
      url: "codes.json?it=" + iteration,
      success: successFunc,
      dataType: 'json',
      error: function() {
        fetchInProgress = false;
        createAlert('danger', 'Oops.', 'Server could not be contacted. Maybe you (or the server) are offline? Reload to try again.', 0);
        return serverConnectionError = true;
      },
      statusCode: {
        401: function() {
          return window.location.reload();
        },
        500: function() {
          fetchInProgress = false;
          createAlert('danger', 'Oops.', 'The server responded with an internal error. Reload to try again.', 0);
          return serverConnectionError = true;
        }
      }
    });
  };

  filterChange = function() {
    var filter;
    filter = $('#filter').val().toLowerCase();
    return $('.otp-item').each(function(idx, el) {
      if ($(el).find('.title').text().toLowerCase().match(filter) === null) {
        return $(el).hide();
      } else {
        return $(el).show();
      }
    });
  };

  initializeApplication = function() {
    $('#keylist').empty();
    $('#filter').bind('keyup', filterChange);
    tick(500, refreshTimerProgress);
    return fetchCodes(iterationCurrent);
  };

  refreshTimerProgress = function() {
    var secondsLeft;
    secondsLeft = timeLeft();
    $('#timer').css('width', (secondsLeft / 30 * 100) + "%");
    if (secondsLeft < 10 && preFetch === null && !serverConnectionError) {
      return fetchCodes(iterationNext);
    }
  };

  tick = function(delay, fkt) {
    return window.setInterval(fkt, delay);
  };

  timeLeft = function() {
    var now;
    now = new Date().getTime();
    return (currentTimeout - now) / 1000;
  };

  updateCodes = function(data) {
    var i, len, ref, token;
    currentTimeout = new Date(data.next_wrap).getTime();
    if (clipboard) {
      clipboard.destroy();
    }
    $('#initLoader').hide();
    $('#keylist').empty();
    ref = data.tokens;
    for (i = 0, len = ref.length; i < len; i++) {
      token = ref[i];
      createOTPItem(token);
    }
    clipboard = new Clipboard('.otp-item', {
      text: function(trigger) {
        return $(trigger).find('.badge').text().replace(' ', '');
      }
    });
    clipboard.on('success', function(e) {
      createAlert('success', 'Success:', 'Code copied to clipboard', 1000);
      return e.blur();
    });
    clipboard.on('error', function(e) {
      return createAlert('danger', 'Oops.', 'Copy to clipboard failed', 2000);
    });
    filterChange();
    delay(timeLeft() * 1000, function() {
      return fetchCodes(iterationCurrent);
    });
    return fetchInProgress = false;
  };

  updatePreFetch = function(data) {
    preFetch = data;
    return fetchInProgress = false;
  };

}).call(this);
