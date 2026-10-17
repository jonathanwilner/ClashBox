import http from 'node:http';
import fs from 'node:fs';
// Usage: node huawei-auth-callback.mjs AUTH_JSON STATUS_JSON [cn|de]
// Start before opening DevEco authorization. Never log callback tokens.
const [authFile,status,region='de']=process.argv.slice(2);
if(!authFile||!status||!['cn','de'].includes(region))throw Error('Usage: node huawei-auth-callback.mjs AUTH_JSON STATUS_JSON [cn|de]');
const setStatus=(state,detail={})=>fs.writeFileSync(status,JSON.stringify({state,time:new Date().toISOString(),...detail}));
async function json(url,options){const r=await fetch(url,options);const b=await r.text();if(!r.ok||!b)throw Error(`Huawei HTTP ${r.status}, response bytes ${b.length}`);return b.trim().startsWith('eyJ')?{ret:{msg:b.trim()}}:JSON.parse(b)}
const server=http.createServer(async(req,res)=>{
try{
const u=new URL(req.url,'http://localhost');const chunks=[];for await(const chunk of req)chunks.push(chunk);const raw=Buffer.concat(chunks).toString();let payload={};try{payload=JSON.parse(raw)}catch{payload=Object.fromEntries(new URLSearchParams(raw))}const token=u.searchParams.get('tempToken')||u.searchParams.get('token')||payload.tempToken||payload.token;
if(!token){res.writeHead(400);res.end('Missing Huawei token. Restart DevEco authorization.');return}
setStatus('exchanging');
const site=u.searchParams.get('site')||payload.site||region.toUpperCase();
const host=site.toUpperCase()==='CN'?'cn.devecostudio.huawei.com':'de.devecostudio.huawei.com';
const base=`https://${host}/authrouter/auth/api/`;
const r=await json(base+'temptoken/check?site='+encodeURIComponent(site)+'&tempToken='+encodeURIComponent(token)+'&appid=1007&version=0.0.0');
const jwtToken=r?.ret?.msg;if(!jwtToken||jwtToken.split('.').length!==3)throw Error('Huawei returned no JWT');
const info=await json(base+'jwToken/check',{headers:{refresh:'false',jwtToken}});
if(!info.userInfo?.accessToken)throw Error('Huawei returned no access token');
const file=authFile;if(fs.existsSync(file))fs.copyFileSync(file,file+'.backup-'+Date.now());fs.writeFileSync(file,JSON.stringify({...info,jwtToken,authRoot:`https://${host}`},null,2));
setStatus('authenticated',{host});res.writeHead(200,{'content-type':'text/html; charset=utf-8'});res.end('<h2>Huawei developer authentication saved successfully.</h2><p>You can return to Codex. Signing can continue.</p>');
}catch(e){setStatus('failed',{error:e instanceof SyntaxError?'Huawei response was not valid JSON':e.message});res.writeHead(502);res.end('Huawei authorization exchange failed. Codex has the diagnostic status.');}
});server.listen(8888,'127.0.0.1',()=>setStatus('listening'));
setTimeout(()=>server.close(()=>process.exit(0)),30*60*1000);
