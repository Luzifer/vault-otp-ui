const iterationCurrent = 'current'
const iterationNext = 'next'

const app = new Vue({

  computed: {
    filteredItems() {
      if (this.filter === '') {
        return this.otpItems
      }

      const items = []

      for (let i of this.otpItems) {
        if (i.name.toLowerCase().match(this.filter.toLowerCase())) {
          items.push(i)
        }
      }

      return items
    },

    minPeriod() {
      let min = 99999

      for (let i of this.otpItems) {
        if (i.period > 0 && i.period < min) {
          min = i.period
        }
      }

      if (min === 99999) {
        min = 30
      }

      return min
    },
  },

  data: {
    authUrl,
    backoff: 500,
    currentTimeout: null,
    fetchInProgress: false,
    filter: '',
    lastFetch: null,
    loading: true,
    preFetch: null,
    signedIn,
    otpItems: [],
    timeLeftPerc: 0.0,
  },

  el: '#application',

  methods: {

    // Let user know whether the copy command was successful
    codeCopyResult(success) {
      this.createAlert(success ? 'success' : 'danger', 'Copy to clipboard...', 'Code copied to clipboard')
    },

    // Wrapper around toast creation
    createAlert(variant, title, text, autoHideDelay=2000) {
      this.$bvToast.toast(text, {
        autoHideDelay,
        title,
        toaster: 'b-toaster-bottom-center',
        variant,
      })
    },

    // Main functionality: Fetch codes, handle errors including backoff
    fetchCodes(iteration=iterationCurrent) {
      if (this.fetchInProgress || !this.signedIn) {
        return
      }

      if (this.lastFetch && this.lastFetch.getTime() + this.backoff > new Date().getTime()) {
        // slow down spammy requests
        return
      }
      this.lastFetch = new Date()

      this.fetchInProgress=true

      let successFunc= iteration == iterationCurrent ? this.updateCodes : this.updatePreFetch
      if (iteration == iterationCurrent && this.preFetch !== null) {
        successFunc(this.preFetch)
        this.fetchInProgress = false
        return
      }

      axios.get(`codes.json?it=${iteration}`)
        .then(resp => {
          successFunc(resp.data)
          this.backoff = 500 // Reset backoff to 500ms
        })
        .catch(err => {
          this.backoff = this.backoff * 1.5 > 30000 ? 30000 : this.backoff * 1.5

          if (err.response && err.response.status) {
            switch (err.response.status) {
              case 401:
                this.createAlert('danger', 'Logged out...', 'Server has no valid token for you: You need to re-login.')
                this.signedIn = false
                this.otpItems = []
                break

              case 500:
                this.createAlert('danger', 'Oops.', `Something went wrong when fetching your codes, will try again in ${Math.round(this.backoff / 1000)}s...`, this.backoff)
                break;
            }
          } else {
            console.error(err)
            this.createAlert('danger', 'Oops.', `The request went wrong, will try again in ${Math.round(this.backoff / 1000)}s...`, this.backoff)
          }

          if (iteration === iterationCurrent) {
            this.otpItems = []
            this.loading = true
          }
        })
        .finally(() => { this.fetchInProgress=false })
    },

    // Format code for better readability: 000 000 or 00 000 000
    formatCode(code) {
      return code
        .replace(/^([0-9]{3})([0-9]{3})$/, '$1 $2') // 6 digits
        .replace(/^([0-9]{2})([0-9]{3})([0-9]{2})$/, '$1 $2 $3') // 7 digits
        .replace(/^([0-9]{2})([0-9]{3})([0-9]{3})$/, '$1 $2 $3') // 8 digits
    },

    // Update timer bar and trigger re-fetch of codes by time remaining
    refreshTimerProgress() {
      const secondsLeft = this.timeLeft()
      this.timeLeftPerc = secondsLeft / this.minPeriod * 100

      if (secondsLeft < 3 && !this.preFetch && this.signedIn) {
        // Do a pre-fetch to provide a seamless experience
        this.fetchCodes(secondsLeft < 0 ? iterationCurrent : iterationNext)
      }
    },

    // Calculate remaining time for the current batch of codes
    timeLeft() {
      if (!this.currentTimeout) {
        return 0
      }

      const now = new Date().getTime()
      return (this.currentTimeout.getTime() - now) / 1000
    },

    // Update displayed codes
    updateCodes(data) {
      this.currentTimeout = new Date(data.next_wrap)
      this.otpItems = data.tokens
      this.loading = false
      this.preFetch = null

      window.setTimeout(this.fetchCodes, this.timeLeft()*1000)
    },

    // Store received data for later usage
    updatePreFetch(data) {
      this.preFetch = data
    },

  },

  // Initialize application
  mounted() {
    window.setInterval(this.refreshTimerProgress, 500)
    this.fetchCodes(iterationCurrent)
  },

})

