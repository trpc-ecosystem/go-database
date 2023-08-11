package redcas

import (
	"trpc.group/trpc-go/trpc-database/goredis/internal/script"
)

// lua script.
const (
	// SetLua set lua script.
	SetLua = `
local key=KEYS[1];
local newValue=ARGV[1];
local checkCAS=tonumber(ARGV[2]);
local duration=tonumber(ARGV[3]);
local oldCAS=0;
local oldRaw=redis.call('getrange', key, 0, 7);
if (oldRaw ~= false) and (#oldRaw >= 8) then
	for k=8,1,-1 do
		oldCAS = string.byte(oldRaw,k) + (oldCAS * 256);
	end
end
if oldCAS ~= checkCAS then
	if oldCAS == 0 then
		return redis.error_reply(string.format("key not found check %d", checkCAS));
	end
	return redis.error_reply(string.format("cas mismatch check %d old %d", checkCAS, oldCAS));
end
if duration > 0 then
	return redis.call('set', key, newValue, 'px', duration);
else
	return redis.call('set', key, newValue);
end
`
	// HSetLua lua script.
	HSetLua = `
local key=KEYS[1];
local field=ARGV[1];
local newValue=ARGV[2];
local checkCAS=tonumber(ARGV[3]);
local oldCAS=0;
local oldRaw=redis.call('hget', key, field);
if (oldRaw ~= false) and (#oldRaw >= 8) then
	for k=8,1,-1 do
		oldCAS = string.byte(oldRaw,k) + (oldCAS * 256);
	end
end
if oldCAS ~= checkCAS then
	return redis.error_reply(string.format("cas mismatch check %d old %d", checkCAS, oldCAS));
end
return redis.call('hset', key, field, newValue);
`
	// MSetLua MSet lua script.
	MSetLua = `
local values={};
local oldValues=redis.call('mget', unpack(KEYS));
for i = 1,#oldValues do
	local newValue=ARGV[i*2-1];
	local checkCAS=tonumber(ARGV[i*2]);
	values[i*2-1]=KEYS[i]
	values[i*2]=newValue
	if (checkCAS ~= -1) then
		local oldRaw=oldValues[i];
		local oldCAS=0;
		if (oldRaw ~= false) and (#oldRaw >= 8) then
			for k=8,1,-1 do
				oldCAS = string.byte(oldRaw,k) + (oldCAS * 256);
			end
		end
		if checkCAS ~= oldCAS then
			return redis.error_reply(string.format("cas mismatch key %q check %d old %d", KEYS[i], checkCAS, oldCAS));
		end
	end
end
return redis.call('mset', unpack(values));
`
	// HMSetLua HMSet lua script.
	HMSetLua = `
local key=KEYS[1];
local fieldLen=#ARGV/3;
local fields={};	
local values={};  
for i = 1,fieldLen do
	local field=ARGV[i*3-2];
	local value=ARGV[i*3-1];
	fields[i]=field;
	values[i*2-1]=field;
	values[i*2]=value;
end
local oldValues=redis.call('hmget', key, unpack(fields));
for i = 1,#oldValues do
	local checkCAS=tonumber(ARGV[i*3]);
	if (checkCAS ~= -1) then
		local oldRaw=oldValues[i];
		local oldCAS=0;
		if (oldRaw ~= false) and (#oldRaw >= 8) then
			for k=8,1,-1 do
				oldCAS = string.byte(oldRaw,k) + (oldCAS * 256);
			end
		end
		if checkCAS ~= oldCAS then
			return redis.error_reply(string.format("cas mismatch field %q check %d old %d", 
				fields[i], checkCAS, oldCAS));
		end
	end
end
return redis.call('hmset', key, unpack(values));
`
)

var (
	// SetScript is set script optimizer.
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists.
	SetScript = script.New(SetLua)
	// HSetScript HSet script optimization.
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists.
	HSetScript = script.New(HSetLua)
	// MSetScript MSet script optimization.
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists.
	MSetScript = script.New(MSetLua)
	// HMSetScript HMSet script optimization.
	// Execute and upload the script through Eval if the server does not exist,
	// and execute it through EvalSha if it exists.
	HMSetScript = script.New(HMSetLua)
)
