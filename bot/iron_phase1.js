'use strict';

Java.perform(function () {
    const TAG = 'IRON_ADVANCED_MONITOR';

    function log(event, data) {
        const entry = Object.assign({
            tag: TAG,
            time: new Date().toISOString(),
            event: event
        }, data || {});
        console.log(JSON.stringify(entry));
    }

    function install(name, callback) {
        try {
            callback();
            log('hook_installed', { name: name });
        } catch (e) {
            log('hook_error', { name: name, error: String(e) });
        }
    }

    // ==========================================
    // 1. SSL Pinning Bypass
    // ==========================================
    install('SSL_Pinning_Bypass', function () {
        try {
            const TrustManagerImpl = Java.use('com.android.org.conscrypt.TrustManagerImpl');
            TrustManagerImpl.verifyChain.implementation = function (untrustedChain, trustAnchorCertificates, host, clientAuth, ocspData, tlsSctData) {
                log('ssl_bypass', { message: 'Intercepted TrustManagerImpl.verifyChain - Bypassed!' });
                return untrustedChain;
            };
        } catch (e) {
            // تجاهل إيلا كان الكلاس غير موجود في هاد الإصدار من أندرويد
        }

        try {
            const CertificatePinner = Java.use('okhttp3.CertificatePinner');
            CertificatePinner.check.overload('java.lang.String', 'java.util.List').implementation = function (hostname, peerCertificates) {
                log('ssl_bypass', { host: hostname, message: 'OkHttp CertificatePinner checked - Bypassed!' });
                return;
            };
        } catch (e) {}
    });

    // ==========================================
    // 2. Cryptographic Key & Plaintext Extraction
    // ==========================================
    install('SecretKeySpec.Extraction', function () {
        const SecretKeySpec = Java.use('javax.crypto.spec.SecretKeySpec');
        SecretKeySpec.$init.overload('[B', 'java.lang.String').implementation = function (keyBytes, algorithm) {
            // تحويل البايتات إلى هيكل أو نص لمعاينة المفتاح
            var keyHex = '';
            for (var i = 0; i < keyBytes.length; i++) {
                var val = keyBytes[i] & 0xff;
                if (val < 16) keyHex += '0';
                keyHex += val.toString(16);
            }
            
            log('crypto_key_extracted', {
                algorithm: algorithm,
                key_length: keyBytes.length,
                key_hex: keyHex
            });

            return this.$init(keyBytes, algorithm);
        };
    });

    // ==========================================
    // 3. Monitor Cipher Operations (Data & Sizes)
    // ==========================================
    install('Cipher.doFinal', function () {
        const Cipher = Java.use('javax.crypto.Cipher');
        const original = Cipher.doFinal.overload('[B');

        original.implementation = function (input) {
            const inputLength = input ? input.length : 0;
            let algorithm = 'unknown';

            try {
                algorithm = String(this.getAlgorithm());
            } catch (_) {}

            log('cipher_start', {
                algorithm: algorithm,
                input_length: inputLength
            });

            try {
                const output = original.call(this, input);
                log('cipher_success', {
                    algorithm: algorithm,
                    output_length: output ? output.length : 0
                });
                return output;
            } catch (e) {
                log('cipher_failure', {
                    algorithm: algorithm,
                    error: String(e)
                });
                throw e;
            }
        };
    });

    // ==========================================
    // 4. Monitor Socket Connections
    // ==========================================
    install('Socket.connect', function () {
        const Socket = Java.use('java.net.Socket');
        const original = Socket.connect.overload('java.net.SocketAddress', 'int');

        original.implementation = function (address, timeout) {
            let addressType = 'null';
            try {
                if (address !== null) {
                    addressType = String(address.getClass().getName());
                }
            } catch (_) {}

            log('socket_connect', {
                address_type: addressType,
                timeout_ms: timeout
            });

            return original.call(this, address, timeout);
        };
    });

    log('ready', { message: 'Iron Advanced Monitor with SSL Bypass & Key Scraping is active.' });
});