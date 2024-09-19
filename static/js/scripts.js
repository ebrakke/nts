function initialize() {
  const urlParams = new URLSearchParams(window.location.search);
  const ncryptParam = urlParams.get('ncrypt');

  if (ncryptParam) {
    const password = prompt('Enter password');
    const sk = NostrIdentitySDK.nip49.decrypt(atob(ncryptParam), password);
    const nsec = NostrIdentitySDK.nip19.nsecEncode(sk);
    localStorage.setItem('nostr_nsec', nsec);
    urlParams.delete('ncrypt');
    const newUrl = window.location.pathname + (urlParams.toString() ? '?' + urlParams.toString() : '');
    window.history.replaceState({}, document.title, newUrl);
    window.location.reload();
    return;
  }

  const storedNsec = localStorage.getItem('nostr_nsec');

  if (storedNsec) {
    try {
      const decodedSk = NostrIdentitySDK.nip19.decode(storedNsec).data;
      window.publicKey = NostrIdentitySDK.getPublicKey(decodedSk);
      console.log('Loaded existing key');
    } catch (error) {
      console.error('Error decoding stored nsec:', error);
      generateNewKey();
    }
  } else {
    generateNewKey();
  }
}

function generateNewKey() {
  const sk = NostrIdentitySDK.generateSecretKey();
  const nsec = NostrIdentitySDK.nip19.nsecEncode(sk);
  localStorage.setItem('nostr_nsec', nsec);
  publicKey = NostrIdentitySDK.getPublicKey(sk);
  console.log('Generated new key');
}

// Call initialize when the script loads
initialize();

/**
 * Decrypts an encrypted string using NIP-44.
 * @param {string} encryptedString - The encrypted string to decrypt.
 * @param {string} sk - The private key for decryption.
 * @param {string} publicKey - The public key associated with the encrypted message.
 * @returns {string} The decrypted message.
 */
function decrypt(encryptedString) {
  const storedNsec = localStorage.getItem('nostr_nsec');
  if (!storedNsec) {
    throw new Error('No secret key found. Please initialize the key first.');
  }

  const sk = NostrIdentitySDK.nip19.decode(storedNsec).data;
  const sharedKey = NostrIdentitySDK.nip44.getConversationKey(sk, publicKey);
  const decrypted = NostrIdentitySDK.nip44.decrypt(encryptedString, sharedKey);
  return decrypted;
}

function getPubkey() {
  return publicKey;
}

function getNsec() {
  return localStorage.getItem('nostr_nsec');
}

/**
 * Generates a title automatically based on the given content.
 * @param {string} content - The content to generate a title from.
 * @param {number} [maxLength=50] - The maximum length of the generated title.
 * @returns {string} The generated title.
 */
function generateTitle(content, maxLength = 50) {
  // Remove any HTML tags
  // Remove special characters, retain only words
  const cleanContent = content.replace(/[^\w\s]/g, '');

  // Split the content into words
  const words = cleanContent.split(/\s+/);

  // Take the first few words
  let title = words.slice(0, 10).join(' ');

  // Truncate if it's too long
  if (title.length > maxLength) {
    title = title.substring(0, maxLength).trim();
    // Ensure we don't cut off in the middle of a word
    const lastSpaceIndex = title.lastIndexOf(' ');
    if (lastSpaceIndex > 0) {
      title = title.substring(0, lastSpaceIndex);
    }
    title += '...';
  }
  console.log(title)

  return title;
}

/**
 * Creates and signs a "note to self" event (kind 1990) with encrypted content and title.
 * @param {string} content - The content of the note.
 * @param {string} [title] - The title of the note. If not provided, it will be generated.
 * @returns {Object} The signed event object.
 */
function createNoteToSelfEvent(content, title = '') {
  const storedNsec = localStorage.getItem('nostr_nsec');
  if (!storedNsec) {
    throw new Error('No secret key found. Please initialize the key first.');
  }

  const sk = NostrIdentitySDK.nip19.decode(storedNsec).data;
  const pubkey = NostrIdentitySDK.getPublicKey(sk);

  // Generate title if not provided
  if (!title) {
    title = generateTitle(content);
  }

  // Encrypt content and title
  const conversationKey = NostrIdentitySDK.nip44.getConversationKey(sk, pubkey);

  const innerEvent = {
    kind: 23,
    created_at: Math.floor(Date.now() / 1000),
    tags: [
      ['title', title]
    ],
    content
  }

  const signedInnerEvent = NostrIdentitySDK.finalizeEvent(innerEvent, sk);

  // Create the event object
  const event = {
    kind: 31234,
    created_at: Math.floor(Date.now() / 1000),
    tags: [
      ['d', `${Math.random().toString(36).substring(2, 15)}`],
      ['k', '23']
    ],
    content: NostrIdentitySDK.nip44.encrypt(JSON.stringify(signedInnerEvent), conversationKey)
  };

  // Sign the event
  const signedEvent = NostrIdentitySDK.finalizeEvent(event, sk);
  return signedEvent;
}

document.addEventListener('DOMContentLoaded', function () {
  // HTMX before request handler
  document.body.addEventListener('htmx:beforeRequest', function (evt) {
    if (evt.detail.requestConfig.path === '/save-note') {
      const content = document.getElementById('note-content').value;

      if (!content.trim()) {
        evt.preventDefault();
        showError('Note content cannot be empty.');
        return;
      }


      try {
        const event = createNoteToSelfEvent(content);

        evt.detail.xhr.setRequestHeader('Content-Type', 'application/json');
        evt.detail.xhr.send(JSON.stringify(event));
        evt.preventDefault();
      } catch (error) {
        evt.preventDefault();
        showError(`Failed to create note: ${error.message}`);
      }
    }
  });
  // HTMX after request handler
  document.body.addEventListener('htmx:afterRequest', function (evt) {
    if (evt.detail.requestConfig.path === '/save-note') {
      if (evt.detail.successful) {
        document.getElementById('note-content').value = '';
        clearError();
      } else {
        showError('Failed to save note. Please try again.');
      }
    }
  });
});


/**
 * Displays an error message to the user.
 * @param {string} message - The error message to display.
 */
function showError(message) {
  const errorElement = document.getElementById('error-message');
  if (errorElement) {
    errorElement.textContent = message;
    errorElement.classList.remove('hidden');
  } else {
    alert(message); // Fallback to alert if error element doesn't exist
  }
}

/**
 * Clears the displayed error message.
 */
function clearError() {
  const errorElement = document.getElementById('error-message');
  if (errorElement) {
    errorElement.textContent = '';
    errorElement.classList.add('hidden');
  }
}

/**
 * Generates a QR code for the given URL.
 * @param {string} url - The URL to encode in the QR code.
 */
function generateQRCode(url) {
  const qrcodeElement = document.getElementById('qrcode');
  qrcodeElement.innerHTML = ''; // Clear any existing QR code

  new QRCode(qrcodeElement, {
    text: url,
    width: 256,
    height: 256,
    colorDark: "#000000",
    colorLight: "#ffffff",
    correctLevel: QRCode.CorrectLevel.H
  });
}

/**
 * Clears the user's key from localStorage after confirmation.
 * Refreshes the page after clearing the key.
 */
function clearKey() {
  if (confirm("Are you sure you want to reset? This action cannot be undone.")) {
    localStorage.removeItem('nostr_nsec');
    location.href = '/';
  }
}

function generateShareUrl(password) {
  const nsec = getNsec();
  console.log(nsec, password);
  if (!nsec) {
    return '';
  }
  const sk = NostrIdentitySDK.nip19.decode(nsec).data;
  const shareUrl = window.location.href + '?ncrypt=' + btoa(NostrIdentitySDK.nip49.encrypt(sk, password));
  console.log(shareUrl);
  return shareUrl;
}

