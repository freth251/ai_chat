import OpenAI from 'openai';
const openai = new OpenAI({
    organization: "org-oJuT4oW5KnhCpbA699qFWcv8",
    apiKey: "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",
});
import express from 'express';
import bcrypt from 'bcrypt';
import pg from 'pg';
import jwt from 'jsonwebtoken'
import axios from 'axios'
const { Pool } = pg;
const secretKey = 'aD&3jD*7d!9Hh12@L&fgq9Eh';
const app = express();
const PORT = 3000;
const MODEL_PORT= 11434;
// Use environment variables or directly provide connection details
const pool = new Pool({
    user: 'amenabshir',
    host: 'localhost',
    database: 'chatbot_db',
    port: 5432
});
app.use(express.json());

app.post('/api/chat', async (req, res) => {
    const bearerHeader = req.headers['authorization'];
    if (typeof bearerHeader !== 'undefined') {
        const bearer = bearerHeader.split(' ');
        const bearerToken = bearer[1];
        
    } else {
        return res.status(500).send('Unauthorized');
    }

    
    
    const userMessage = req.body.message;

    try {
        const response = await axios.post('http://127.0.0.1:11434/api/generate', {
            model: 'mistral',
            prompt: userMessage,
        });

        console.log("Returned from model: ", response)
        const aiMessage = response.data.response;
        const messages=[userMessage, aiMessage]
        pool.query('UPDATE chat_history SET messages = array_append(messages, $2) WHERE user_id=$1',
        [auth["id"], messages],)
        res.json({ message: aiMessage });

    } catch (error) {
        console.error('Error calling AI API:', error);
        res.status(500).send('Error interacting with AI');
    }
});
app.post('/register', async (req, res) => {
      console.log("received register")
      const { username, email, password } = req.body;
  
      try {
        const response = await axios.post('http://127.0.0.1:9400/register', {
            username: username, 
            email: email,
            password: password
        });
        
        res.json({token: response.data.jwt})

    }catch (err) {
        console.log(err)
        res.status(500).send('Server error');
    }
});
  
app.post('/login', async (req, res) => {
    try {
      console.log("received login")
      const { email, password } = req.body;
      const response = await axios.post('http://127.0.0.1:9400/login', {
            email: email,
            password: password
        });
        console.log(response.data)
        res.json({token: response.data.jwt})
    } catch (err) {
      res.status(500).send('Server error');
    }
});

app.post('/history', async (req, res) =>{
    const bearerHeader = req.headers['authorization'];
    if (typeof bearerHeader !== 'undefined') {
        const bearer = bearerHeader.split(' ');
        const bearerToken = bearer[1];
        console.log('token: ' + bearerToken)
        var auth= jwt.verify(bearerToken, secretKey, (err, authData) => {
            if (err) {
                console.log("Unauthorized")
                return ;
            }
            return authData
        });
        if (!(auth)){
            return res.status(500).send('Unauthorized');
        }
    } else {
        return res.status(500).send('Unauthorized');
    }
    try{
        console.log(auth["id"])
        const history= await pool.query('SELECT messages FROM chat_history WHERE user_id=$1', [auth["id"]])
        console.log(history.rows[0])
        res.json({history: history.rows[0]})

    } catch (err) {
        res.status(500).send('Server error')
    }
});
app.use(express.static('public'));

app.listen(PORT, () => {
    console.log(`Server is running on http://localhost:${PORT}`);
});
