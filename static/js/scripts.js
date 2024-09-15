function initialize() {
  const storedNsec = localStorage.getItem('nostr_nsec');
  
  if (storedNsec) {
    try {
      const decodedSk = NostrTools.nip19.decode(storedNsec).data;
      window.publicKey = NostrTools.getPublicKey(decodedSk);
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
  const sk = NostrTools.generateSecretKey();
  const nsec = NostrTools.nip19.nsecEncode(sk);
  localStorage.setItem('nostr_nsec', nsec);
  publicKey = NostrTools.getPublicKey(sk);
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

  const sk = NostrTools.nip19.decode(storedNsec).data;
  const sharedKey = NostrTools.nip44.getConversationKey(sk, publicKey);
  const decrypted = NostrTools.nip44.decrypt(encryptedString, sharedKey);
  return decrypted;
}

function getPubkey() {
  return publicKey;
}

/**
 * Generates a title automatically based on the given content.
 * @param {string} content - The content to generate a title from.
 * @param {number} [maxLength=50] - The maximum length of the generated title.
 * @returns {string} The generated title.
 */
function generateTitle(content, maxLength = 50) {
  // Remove any HTML tags
  const cleanContent = content.replace(/<[^>]*>/g, '');
  
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

  const sk = NostrTools.nip19.decode(storedNsec).data;
  const pubkey = NostrTools.getPublicKey(sk);

  // Generate title if not provided
  if (!title) {
    title = generateTitle(content);
  }

  // Encrypt content and title
  const conversationKey = NostrTools.nip44.getConversationKey(sk, pubkey);
  const encryptedContent = NostrTools.nip44.encrypt (content, conversationKey);
  const encryptedTitle = NostrTools.nip44.encrypt(title, conversationKey);

  // Create the event object
  const event = {
    kind: 1990,
    created_at: Math.floor(Date.now() / 1000),
    tags: [],
    content: encryptedContent
  };

  // Add encrypted title as a tag
  event.tags.push(['title', encryptedTitle]);

  // Sign the event
  const signedEvent = NostrTools.finalizeEvent(event, sk);

  return signedEvent;
}

// HTMX before request handler
document.body.addEventListener('htmx:beforeRequest', function(evt) {
  if (evt.detail.elt.id === 'save-note-form') {
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
    } catch (error) {
      evt.preventDefault();
      showError(`Failed to create note: ${error.message}`);
    }
  }
  console.log('event', event);
});

// HTMX after request handler
document.body.addEventListener('htmx:afterRequest', function(evt) {
  if (evt.detail.elt.id === 'save-note-form') {
    if (evt.detail.successful) {
      document.getElementById('note-content').value = '';
      document.getElementById('note-title').value = '';
      clearError();
    } else {
      showError('Failed to save note. Please try again.');
    }
  }
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


