



document.addEventListener("DOMContentLoaded", async () => {
    const registerForm = document.getElementById("register-form");
    const loginForm = document.getElementById("login-form");
    const chatOutput = document.getElementById('chat-output');
    const chatInput = document.getElementById('chat-input');
    const button = document.getElementById('input-button')
    if (registerForm) {
        registerForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            const username = document.getElementById("username").value;
            const email = document.getElementById("email").value;
            const password = document.getElementById("password").value;
            const confirmPassword = document.getElementById("confirm-password").value;

            console.log(password)
            if (password !== confirmPassword) {
                alert("Passwords do not match!");
                return;
            }

            // Send a request to the backend
            const response = await fetch("/register", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({ username, email, password })
            });

            const data = await response.json();

            if (response.status === 200) {
                alert("Registered successfully!");
                window.location.href = "login.html";
            } else {
                alert(data.error || "Registration failed");
            }
        });
    }

    if (loginForm) {
        loginForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            const email = document.getElementById("login-email").value;
            const password = document.getElementById("login-password").value;

            // Send a request to the backend
            const response =  fetch("/login", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({ email, password })
            })
            .then(response => response.json())
            .then(data => {
                localStorage.setItem('token', data.token);
                window.location.href = "index.html";
          
            });
        });
    }
    if (chatOutput || chatInput || button){
        if (localStorage.getItem('firstLoadDone') === null) {
            const token = localStorage.getItem('token');
            const response = await fetch("/history", {
                method: "POST",
                headers: {
                    'Authorization': 'Bearer ' + token,
                    "Content-Type": "application/json"
                },
            });

            const data = await response.json();
            console.log(data)
            const extractedArrays = data.history.messages.map(message => {
                if (message.startsWith('{') && message.endsWith('}')) {
                  return message.slice(1, -1).split(',').map(item => item.replace(/"/g, ''));
                }
                return null; // or return the original message if you prefer
              });
            extractedArrays.forEach((item, index) => {
                if (item==null) return;
                else if (item.length==2){
                    const userMessageDiv = document.createElement('div');
                    const aiMessageDiv = document.createElement('div');
                    appendMessage('User', item[0], userMessageDiv, chatOutput);
                    appendMessage('AI', item[1], aiMessageDiv, chatOutput )
                }
            });
            localStorage.setItem('firstLoadDone', true);
        }
        if (chatInput){
            chatInput.addEventListener('keydown', async (event) => {
                if (event.key === 'Enter') {
                    chatAi(chatInput, chatOutput)
                }
            });
        }
        if (button){
            button.addEventListener('click', async function() {
                // Code to be executed when the button is clicked
                chatAi(chatInput, chatOutput)
            });
        }

    }
});
async function chatAi(chatInput, chatOutput){
    const userMessage = chatInput.value.trim();
        chatInput.value = '';

        if (userMessage) {
            const userMessageDiv = document.createElement('div');
            appendMessage('User', userMessage, userMessageDiv, chatOutput);
            
            const aiMessageDiv = document.createElement('div');
            appendMessage('AI', '...', aiMessageDiv, chatOutput )
            const token = localStorage.getItem('token');
            const response = await fetch('/api/chat', {
                method: 'POST',
                headers: {
                    'Authorization': 'Bearer ' + token,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ message: userMessage })
            });

            const data = await response.json();
            appendMessage('AI', data.message, aiMessageDiv, chatOutput);
        }
}
function appendMessage(sender, message, messageDiv, chatOutput) {
    if (message=='...'){
        messageDiv.classList.add("ai-output")
        messageDiv.textContent = `${message}`;
        chatOutput.appendChild(messageDiv);
        chatOutput.scrollTop = chatOutput.scrollHeight;
    }
    else if (sender=='AI'){
        messageDiv.classList.add("ai-output")
        messageDiv.textContent = ``;
        if (localStorage.getItem('firstLoadDone') === null){
            messageDiv.textContent = `${message}`;
            chatOutput.appendChild(messageDiv);
            chatOutput.scrollTop = chatOutput.scrollHeight;
        }else{
            typeMessage(message, messageDiv, chatOutput)
        }
       
    }
    else if (sender=='User'){
        messageDiv.classList.add("user-output")
        messageDiv.textContent = `${message}`;
        chatOutput.appendChild(messageDiv);
        chatOutput.scrollTop = chatOutput.scrollHeight;
    }
}
function typeMessage(message, element, chatOutput) {
    let index = 0;
    
    // Start the typing effect
    const interval = setInterval(() => {
        // Add one character to the element's text content
        element.textContent += message[index];
        index++;
        chatOutput.scrollTop = chatOutput.scrollHeight;

        // If the entire message is displayed, clear the interval
        if (index === message.length) {
            clearInterval(interval);
        }
    }, 50); // 100 milliseconds delay between each character; adjust as needed
}
