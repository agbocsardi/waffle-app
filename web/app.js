let currentUser = null;

async function joinConversation() {
    const inviteCode = document.getElementById('invite-code').value;
    const username = document.getElementById('username').value;
    
    try {
        const response = await fetch('/api/conversations/join', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ invite_code: inviteCode, username: username })
        });
        
        if (response.ok) {
            currentUser = username;
            document.getElementById('auth-section').classList.add('hidden');
            document.getElementById('app-section').classList.remove('hidden');
            loadConversations();
        } else {
            alert('Failed to join conversation');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error joining conversation');
    }
}

async function createConversation() {
    const name = document.getElementById('conversation-name').value;
    
    try {
        const response = await fetch('/api/conversations', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ name: name })
        });
        
        if (response.ok) {
            const data = await response.json();
            alert(`Conversation created! Invite code: ${data.invite_code}`);
            loadConversations();
        } else {
            alert('Failed to create conversation');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error creating conversation');
    }
}

async function loadConversations() {
    try {
        const response = await fetch('/api/conversations');
        
        if (response.ok) {
            const conversations = await response.json();
            const conversationsList = document.getElementById('conversations-list');
            conversationsList.innerHTML = '';
            
            conversations.forEach(conv => {
                const div = document.createElement('div');
                div.textContent = `${conv.name} (${conv.id})`;
                conversationsList.appendChild(div);
            });
        }
    } catch (error) {
        console.error('Error loading conversations:', error);
    }
}

async function uploadVideo() {
    const fileInput = document.getElementById('video-file');
    const conversationId = document.getElementById('conversation-id').value;
    
    if (fileInput.files.length === 0) {
        alert('Please select a video file');
        return;
    }
    
    const formData = new FormData();
    formData.append('file', fileInput.files[0]);
    formData.append('conversation_id', conversationId);
    
    try {
        const response = await fetch('/api/upload', {
            method: 'POST',
            body: formData
        });
        
        if (response.ok) {
            alert('Video uploaded successfully!');
            loadVideos(conversationId);
        } else {
            alert('Failed to upload video');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error uploading video');
    }
}

async function loadVideos(conversationId) {
    try {
        const response = await fetch(`/api/videos?conversation_id=${conversationId}`);
        
        if (response.ok) {
            const videos = await response.json();
            const videosList = document.getElementById('videos-list');
            videosList.innerHTML = '';
            
            videos.forEach(video => {
                const div = document.createElement('div');
                div.className = 'video-item';
                div.innerHTML = `
                    <p>Uploaded by: ${video.uploader}</p>
                    <p>Status: ${video.status}</p>
                    <p>Date: ${new Date(video.uploaded_at).toLocaleString()}</p>
                `;
                videosList.appendChild(div);
            });
        }
    } catch (error) {
        console.error('Error loading videos:', error);
    }
}