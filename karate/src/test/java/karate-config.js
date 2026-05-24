function fn() {
    var config = {};
    config.baseUrl = karate.properties['baseUrl'] || 'http://localhost:8080';
    return config;
}
