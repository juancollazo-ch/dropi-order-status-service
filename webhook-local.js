// Servidor webhook local simple para recibir y mostrar webhooks
const http = require('http');
const fs = require('fs');

const PORT = 3000;
const LOG_FILE = 'webhooks-recibidos.json';

// Crear archivo de log si no existe
if (!fs.existsSync(LOG_FILE)) {
    fs.writeFileSync(LOG_FILE, '[]');
}

const server = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/webhook') {
        let body = '';

        req.on('data', chunk => {
            body += chunk.toString();
        });

        req.on('end', () => {
            try {
                const data = JSON.parse(body);
                const timestamp = new Date().toISOString();
                
                console.log('\n========================================');
                console.log('🔔 WEBHOOK RECIBIDO:', timestamp);
                console.log('========================================');
                console.log('Orden ID:', data.id);
                console.log('Estado Actual:', data.status);
                
                if (data.history && data.history.length >= 2) {
                    const ultimo = data.history[data.history.length - 1];
                    const penultimo = data.history[data.history.length - 2];
                    console.log('Estado Anterior:', penultimo.status);
                    console.log('Cambio:', penultimo.status, '→', ultimo.status);
                }
                
                console.log('\nDatos completos:');
                console.log(JSON.stringify(data, null, 2));
                console.log('========================================\n');

                // Guardar en archivo
                const logs = JSON.parse(fs.readFileSync(LOG_FILE));
                logs.push({
                    timestamp,
                    order_id: data.id,
                    status: data.status,
                    data: data
                });
                fs.writeFileSync(LOG_FILE, JSON.stringify(logs, null, 2));

                // Responder OK
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ success: true, message: 'Webhook recibido' }));
            } catch (error) {
                console.error('Error procesando webhook:', error);
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ success: false, error: error.message }));
            }
        });
    } else if (req.method === 'GET' && req.url === '/') {
        // Página de inicio
        res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
        res.end(`
            <!DOCTYPE html>
            <html>
            <head>
                <title>Webhook Receiver</title>
                <meta charset="utf-8">
                <style>
                    body { font-family: Arial; max-width: 1200px; margin: 50px auto; padding: 20px; }
                    h1 { color: #333; }
                    .status { padding: 10px; background: #e8f5e9; border-left: 4px solid #4caf50; margin: 10px 0; }
                    pre { background: #f5f5f5; padding: 15px; overflow-x: auto; }
                    .webhook { border: 1px solid #ddd; padding: 15px; margin: 10px 0; border-radius: 5px; }
                    .timestamp { color: #666; font-size: 0.9em; }
                </style>
            </head>
            <body>
                <h1>🔔 Webhook Receiver Local</h1>
                <div class="status">
                    ✅ Servidor corriendo en http://localhost:${PORT}/webhook
                </div>
                <p>Los webhooks recibidos se mostrarán en la consola y se guardarán en <code>webhooks-recibidos.json</code></p>
                <p><strong>Endpoint:</strong> <code>http://localhost:${PORT}/webhook</code></p>
                <p><strong>Método:</strong> POST</p>
                <p><strong>Webhooks recibidos:</strong> Ver archivo <code>webhooks-recibidos.json</code></p>
            </body>
            </html>
        `);
    } else {
        res.writeHead(404);
        res.end('Not Found');
    }
});

server.listen(PORT, () => {
    console.log('========================================');
    console.log('🚀 Servidor Webhook Local Iniciado');
    console.log('========================================');
    console.log(`📍 URL: http://localhost:${PORT}/webhook`);
    console.log(`📊 Ver en navegador: http://localhost:${PORT}`);
    console.log(`📝 Logs guardados en: ${LOG_FILE}`);
    console.log('========================================');
    console.log('Esperando webhooks...\n');
});
