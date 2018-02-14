AppConfig.guest = false;

var app = angular.module('souris-app', ['souris-common']);
app.config(['$routeProvider', function(route) {
  route.
    when('/',                            HOME).

    when('/profile',                  PROFILE).
    when('/aspect/:aspect',           PROFILE).
    when('/user/:ident',              PROFILE).

    when('/admin',                     ASPECT).
    when('/admin/:aspect',             ASPECT).
    when('/admin/:aspect/group/:group', GROUP).
    when('/admin/:aspect/hash/:name',    HASH).

    when('/oauth',                      OAUTH).

    when('/peer',                       PEERS).
    when('/peer/:id',                   PEERS).

    otherwise({redirectTo:'/'});
}]);

function menu_items(self, active) {
    return {title: "", items: [{link: '/',      title: "Home", active: "home" === active}]};
}


var HOME = {
    templateUrl: '/ui/home.html',
    controller: ['$scope', function(self) {}]
};
var PROFILE = {
    templateUrl: '/ui/profile.html',
    controller: ['$scope', '$routeParams', '$sourisRemoteService', function(self, args, req) {
        if (!self.$user.active) return;

        self.ident = self.$user.ident;
        self.aspect = self.$user.aspect;

        var loadProfile = function(u) {
            if (u.site !== undefined && u.site.mail !== undefined)
                u.gravatar = hashMD5(u.site.mail);
            self.user = u;
        };

        if (args.ident !== undefined) self.ident = args.ident;
        if (args.aspect !== undefined) self.aspect = args.aspect;

         if (args.ident !== undefined)
            req('/v1/profile/user.profile(:ident)',self.aspect)
                .get({ident: self.ident})
                .success(loadProfile);

        else if (args.aspect !== undefined)
            req('/v1/profile/user.profile',self.aspect)
                .get()
                .success(loadProfile);

        else loadProfile(self.$user);

        self.$save = function(aspect, ident, site_txt, app_txt, local_txt) {
            console.log(site_txt);
            console.log(app_txt);
            console.log(local_txt);

            var o = {
                site:  parseConfig(site_txt),
                app:   parseConfig(app_txt),
                local: parseConfig(local_txt)
            };

            req('/v1/profile/user.profile')
                .put({aspect: aspect}, o)
                .success(loadProfile);
        };
    }]
};
var ASPECT = {
    templateUrl: '/ui/aspect-editor.html',
    controller: ['$scope', '$routeParams', '$sourisRemoteService', '$location',
    function(self, args, req, loc) {
        if (!self.$user.active) return;

        self.aspect = args.aspect;
        //if (self.aspect === undefined) self.aspect = 'default';
        self.active_aspect = self.aspect;

        var load = function(aspect) {
            req("/v1/profile/aspect.list", aspect).get().success(function (d) {
                if (self.aspect !== undefined) {
                    var ok = false;
                    for (var i = 0; i < d.length; i++) if (d[i] === self.aspect) ok = true;
                    if (!ok) d.push(self.aspect);
                }
                self.aspectList = d;
            });
            if (aspect === undefined) return;
            req("/v1/profile/aspect.group(:aspect)", aspect).
                get({aspect: aspect}).
                success(function (d) { self.groupList = d;}).
                error(function(){ self.groupList = []; });

            req('/v1/profile/hash.list(:aspect)', aspect).
                get({aspect: aspect}).
                success(function(d){ self.hashList = d; }).
                error(function(){ self.hashList = []; });
        }; load(self.aspect);

        self.addAspect = function(a) {
            loc.path("/admin/"+a);
        };
        self.addHash = function(a, h) {
            loc.path("/admin/"+a+"/hash/"+h);
        };
        self.addGroup = function(a, g) {
            loc.path("/admin/"+a+"/group/"+g);
        };
    }]
};
var GROUP = {
    templateUrl: '/ui/aspect-editor.html',
    controller: ['$scope', '$routeParams', '$sourisRemoteService', '$location',
    function(self, args, req, loc) {
        if (!self.$user.active) return;

        self.aspect = args.aspect;
        self.group = args.group;
        self.active_aspect = self.aspect;
        self.active_group = self.group;

        var load = function(aspect) {
            req("/v1/profile/aspect.list", aspect).get().success(function (d) {
                if (self.aspect !== undefined) {
                    var ok = false;
                    for (var i = 0; i < d.length; i++) if (d[i] === self.aspect) ok = true;
                    if (!ok) d.push(self.aspect);
                }
                self.aspectList = d;
            });

            req("/v1/profile/aspect.group(:aspect)", aspect).
               get({aspect: aspect}).
                success(function (d) {
                    if (self.group !== undefined) {
                        var ok = false;
                        for (var i = 0; i < d.length; i++) if (d[i] === self.group) ok = true;
                        if (!ok) d.push(self.group);
                    }
                    self.groupList = d;
                }).
                error(function(){
                    var d=[];
                    if (self.group !== undefined) {
                        var ok = false;
                        for (var i = 0; i < d.length; i++) if (d[i] === self.group) ok = true;
                        if (!ok) d.push(self.group);
                    }
                    self.groupList = d;
                });

            req('/v1/profile/hash.list(:aspect)', aspect).
                get({aspect: aspect}).
                success(function(d){ self.hashList = d; }).
                error(function(){ self.hashList = []; });
        }; load(self.aspect);

        var loadUsers = function(aspect, group) {
            req('/v1/profile/group.users(:aspect,:group)', aspect).
                get({aspect: aspect, group: group}).
                    success(function(d){ self.users = d; self.user_txt = d.join("\n"); }).
                    error(function(){ self.users = []; self.user_txt = ""; });
        }; loadUsers(args.aspect, args.group);

        var loadRoles = function(aspect, group) {
            req('/v1/profile/group.roles(:aspect,:group)', aspect).
                get({aspect: aspect, group: group}).
                success(function(d){ self.roles = d; self.role_txt = d.join("\n"); }).
                error(function(){ self.roles = []});
        }; loadRoles(args.aspect, args.group);

        self.addHash = function(a, h) {
            loc.path("/admin/"+a+"/hash/"+h);
        };
        self.addGroup = function(a, g) {
            loc.path("/admin/"+a+"/group/"+g);
        };

        self.$saveUser = function(aspect, group, users) {
            var v = self.users;
            var u = users.split("\n");

            for (var i=0; i<v.length; i++) if (v[i] !== "") if (u.indexOf(v[i]) === -1) {
                console.log("rem: " + v[i]);

                req('/v1/profile/group.user(:aspect,:group,:user)', aspect).delete({
                    aspect: aspect,
                    group: group,
                    user: v[i]
                }).error(function(e){ self.error = e; });
            }

            for (var i=0; i<u.length; i++) if (u[i] !== "") if (v.indexOf(u[i]) === -1) {
                console.log("add: " + u[i]);

                req('/v1/profile/group.user(:aspect,:group,:user)', aspect).put({
                    aspect: aspect,
                    group: group,
                    user: u[i]
                }).error(function(e){ self.error = e;});
            }

            self.users = u;
            self.user_txt = users;
        };
        self.$saveRole = function(aspect, group, roles) {
            var v = self.roles;
            var u = roles.split("\n");
            var s;

            for (var i=0; i<u.length; i++) if (v[i] !== "") if (v.indexOf(u[i]) === -1) {
                console.log("add: " + u[i]);

                s = u[i].split("\/");
                if (s.length !== 2) continue;
                req('/v1/profile/group.role(:aspect,:group,:assign,:role)', s[0]).put({
                    aspect: s[0],
                    group: group,
                    assign: s[0],
                    role: s[1]
                }).error(function(e){ self.error = e;});
            }

            for (var i=0; i<v.length; i++) if (v[i] !== "") if (u.indexOf(v[i]) === -1) {
                console.log("rem: " + v[i]);

                s = v[i].split("\/");
                if (s.length !== 2) continue;
                req('/v1/profile/group.role(:aspect,:group,:assign,:role)', s[0]).delete({
                    aspect: s[0],
                    group: group,
                    assign: s[0],
                    role: s[1]
                }).error(function(e){ self.error = e;});
            }

            self.roles = u;
            self.role_txt = roles;
        };
    }]
};
var HASH = {
    templateUrl: '/ui/aspect-editor.html',
    controller: ['$scope', '$routeParams', '$sourisRemoteService', '$location',
    function(self, args, req, loc) {
        if (!self.$user.active) return;

        self.aspect = args.aspect;
        self.name = args.name;
        self.active_aspect = self.aspect;
        self.active_hash = self.name;

        var load = function(aspect) {
            req("/v1/profile/aspect.list", aspect).get().success(function (d) {
                if (self.aspect !== undefined) {
                    var ok = false;
                    for (var i = 0; i < d.length; i++) if (d[i] === self.aspect) ok = true;
                    if (!ok) d.push(self.aspect);
                }
                self.aspectList = d;
            });

            req("/v1/profile/aspect.group(:aspect)", aspect).
                get({aspect: aspect}).
                success(function (d) {
                    if (self.group !== undefined) {
                        var ok = false;
                        for (var i = 0; i < d.length; i++) if (d[i] === self.group) ok = true;
                        if (!ok) d.push(self.group);
                    }
                    self.groupList = d;
                }).
                error(function(){
                    var d=[];
                    if (self.group !== undefined) {
                        var ok = false;
                        for (var i = 0; i < d.length; i++) if (d[i] === self.group) ok = true;
                        if (!ok) d.push(self.group);
                    }
                    self.groupList = d;
                    });

            req('/v1/profile/hash.list(:aspect)', aspect).
                get({aspect: aspect}).
                success(function(d){
                    if (self.name !== undefined) {
                        var ok = false;
                        for (var i = 0; i < d.length; i++) if (d[i] === self.name) ok = true;
                        if (!ok) d.push(self.name);
                    }
                    self.hashList = d;
                }).
                error(function(){
                    var d=[];
                    if (self.name !== undefined) {
                        var ok = false;
                        for (var i = 0; i < d.length; i++) if (d[i] === self.name) ok = true;
                        if (!ok) d.push(self.name);
                    }
                    self.hashList = d;
                });
        }; load(self.aspect);

        var loadHash = function(aspect, name) {
            req('/v1/profile/hash.map(:aspect,:name)', aspect).
                get({aspect: aspect, name: name}).
                success(function(d){
                    self.hash = d;
                    self.hash_txt = encodeConfig(d);
                }).error(function(){ self.hash = {}; self.hash_txt = ""; });
        }; loadHash(args.aspect, args.name);

        self.addHash = function(a, h) {
            loc.path("/admin/"+a+"/hash/"+h);
        };
        self.addGroup = function(a, g) {
            loc.path("/admin/"+a+"/group/"+g);
        };

        self.$save = function(aspect, name, values) {
            var v = parseConfig(values);

            req('/v1/profile/hash.map(:aspect,:name)', aspect).
                put({aspect: aspect, name: name}, v).
                    success(function(){ loadHash(aspect, name); }).
                    error(function(e){ self.error = e;});
        };
        self.$delete = function(aspect, name) {
            req('/v1/profile/hash.map(:aspect,:name)', aspect).
                delete({aspect: aspect, name: name}).
                success(function(){ loc.path('/admin/' + aspect); }).
                error(function(e){ self.error = e;});

            return false;
        };
    }]
};

var OAUTH_ARGS = undefined;
var OAUTH = {
    templateUrl: '/ui/oauth.html',
    reloadOnSearch: false,
    controller: ['$scope', '$routeParams', '$sourisRemoteService', '$location',
    function(self, args, req, location) {
        if(!self.$user.active) return;

        if (OAUTH_ARGS === undefined) {
            OAUTH_ARGS = {
                client_id: args.client_id,
                redirect_uri: args.redirect_uri,
                state: args.state,
                response_type: args.response_type
            };
            location.search({});
        }

        if (OAUTH_ARGS.client_id === undefined)
            location.url("/");

        req('/v1/profile/oauth.authorize','oauth').
            get({client_id: OAUTH_ARGS.client_id}).
            success(function(d){ self.oauth = d; });

        self.$authorize = function() {
            req('/v1/profile/oauth.authorize','oauth').
                post({}, OAUTH_ARGS).
                success(function(d){
                    window.location = OAUTH_ARGS.redirect_uri + "?code=" + d.code + "&state=" + encodeURI(OAUTH_ARGS.state);
                })
        };
    }]
};
var PEERS = {
    templateUrl: '/ui/peers.html',
    controller: ['$scope', '$routeParams', '$sourisRemoteService', '$location', function(self, args, req, loc) {
        if (!self.$user.active) return;

        var user = 'anon';
        var owner = 'anon';
        if (self.$user.ident !== undefined) { user = self.$user.ident; owner = self.$user.ident; }
        if (self.$user.display_name !== undefined) user = self.$user.display_name;


        self.TYPE = ["openvpn","fastd","gre/ipsec","gre/plain","l2tp","pptp","tinc","wireguard","zerotier","other"];
        self.FAMILY = {1:"ipv4", 2:"ipv6", 3:"both"};

        var setNode = function(d) {
            self.error = undefined;
            d.peer_family = "" + d.peer_family;
            d.peer_type = d.peer_type.split(",");

            self.node = d;
            self.node_txt = encodeConfig(d);
        };
        var setNodes = function(d) {
            self.error = undefined;
            self.nodes = d;
        };
        var setError = function(e) {
            self.error = e;
            self.saved = false;
        };
        var setSaved = function(d) {
            self.error = undefined;
            self.saved = true;

            var found=false;
            for (var i=0; i<self.nodes.length; i++) {
                if (self.nodes[i].peer_id === d.peer_id) found=true;
            }
            if (!found) self.nodes.push(d);

            setNode(d);
        };
        var setDeleted = function() {
            loc.path("/peer");
        };

        self.newNode = function() {
            setNode({
                peer_name: ""
            ,   peer_nick: user
            ,   peer_owner: owner
            ,   peer_country: "XD"
            ,   peer_type: "openvpn"
            ,   peer_family: 3
            ,   peer_note: ""
            });
        };
        self.getNode = function(id) {
            req('/v1/peers/peer.node(:id)', "peers")
                .get({id: id})
                .success(setNode)
                .error(setError);
        };
        self.saveNode = function(node) {
            node.peer_family = parseInt(node.peer_family);
            node.peer_type = node.peer_type.join(",");

            if (node.peer_id === undefined)
                req('/v1/peers/peer.nodes', "peers")
                    .post({}, node)
                    .success(setSaved)
                    .error(setError);
            else
                req('/v1/peers/peer.node(:id)', "peers")
                    .put({id: node.peer_id}, node)
                    .success(setSaved)
                    .error(setError);
        };
        self.deleteNode = function(node) {
            if (node.peer_id === undefined)
                self.newNode();
            else
                req('/v1/peers/peer.node(:id)', "peers")
                    .delete({id: node.peer_id}, {})
                    .success(setDeleted)
                    .error(setError);
        };

        req('/v1/peers/peer.nodes', 'peers').get()
            .success(setNodes)
            .error(setError);

        self.peer_id = args.id;

        if (args.id !== undefined)
            self.getNode(args.id);
        else self.newNode();
    }]
};

var hashMD5 = function(e) {
    function h(a, b) {
        var c, d, e, f, g;
        e = a & 2147483648;
        f = b & 2147483648;
        c = a & 1073741824;
        d = b & 1073741824;
        g = (a & 1073741823) + (b & 1073741823);
        return c & d ? g ^ 2147483648 ^ e ^ f : c | d ? g & 1073741824 ? g ^ 3221225472 ^ e ^ f : g ^ 1073741824 ^ e ^ f : g ^ e ^ f
    }

    function k(a, b, c, d, e, f, g) {
        a = h(a, h(h(b & c | ~b & d, e), g));
        return h(a << f | a >>> 32 - f, b)
    }

    function l(a, b, c, d, e, f, g) {
        a = h(a, h(h(b & d | c & ~d, e), g));
        return h(a << f | a >>> 32 - f, b)
    }

    function m(a, b, d, c, e, f, g) {
        a = h(a, h(h(b ^ d ^ c, e), g));
        return h(a << f | a >>> 32 - f, b)
    }

    function n(a, b, d, c, e, f, g) {
        a = h(a, h(h(d ^ (b | ~c), e), g));
        return h(a << f | a >>> 32 - f, b)
    }

    function p(a) {
        var b = "",
            d = "",
            c;
        for (c = 0; 3 >= c; c++) d = a >>> 8 * c & 255, d = "0" + d.toString(16), b += d.substr(d.length - 2, 2);
        return b
    }
    var f = [],
        q, r, s, t, a, b, c, d;
    e = function(a) {
        a = a.replace(/\r\n/g, "\n");
        for (var b = "", d = 0; d < a.length; d++) {
            var c = a.charCodeAt(d);
            128 > c ? b += String.fromCharCode(c) : (127 < c && 2048 > c ? b += String.fromCharCode(c >> 6 | 192) : (b += String.fromCharCode(c >> 12 | 224), b += String.fromCharCode(c >> 6 & 63 | 128)), b += String.fromCharCode(c & 63 | 128))
        }
        return b
    }(e);
    f = function(b) {
        var a, c = b.length;
        a = c + 8;
        for (var d = 16 * ((a - a % 64) / 64 + 1), e = Array(d - 1), f = 0, g = 0; g < c;) a = (g - g % 4) / 4, f = g % 4 * 8, e[a] |= b.charCodeAt(g) << f, g++;
        a = (g - g % 4) / 4;
        e[a] |= 128 << g % 4 * 8;
        e[d - 2] = c << 3;
        e[d - 1] = c >>> 29;
        return e
    }(e);
    a = 1732584193;
    b = 4023233417;
    c = 2562383102;
    d = 271733878;
    for (e = 0; e < f.length; e += 16) q = a, r = b, s = c, t = d, a = k(a, b, c, d, f[e + 0], 7, 3614090360), d = k(d, a, b, c, f[e + 1], 12, 3905402710), c = k(c, d, a, b, f[e + 2], 17, 606105819), b = k(b, c, d, a, f[e + 3], 22, 3250441966), a = k(a, b, c, d, f[e + 4], 7, 4118548399), d = k(d, a, b, c, f[e + 5], 12, 1200080426), c = k(c, d, a, b, f[e + 6], 17, 2821735955), b = k(b, c, d, a, f[e + 7], 22, 4249261313), a = k(a, b, c, d, f[e + 8], 7, 1770035416), d = k(d, a, b, c, f[e + 9], 12, 2336552879), c = k(c, d, a, b, f[e + 10], 17, 4294925233), b = k(b, c, d, a, f[e + 11], 22, 2304563134), a = k(a, b, c, d, f[e + 12], 7, 1804603682), d = k(d, a, b, c, f[e + 13], 12, 4254626195), c = k(c, d, a, b, f[e + 14], 17, 2792965006), b = k(b, c, d, a, f[e + 15], 22, 1236535329), a = l(a, b, c, d, f[e + 1], 5, 4129170786), d = l(d, a, b, c, f[e + 6], 9, 3225465664), c = l(c, d, a, b, f[e + 11], 14, 643717713), b = l(b, c, d, a, f[e + 0], 20, 3921069994), a = l(a, b, c, d, f[e + 5], 5, 3593408605), d = l(d, a, b, c, f[e + 10], 9, 38016083), c = l(c, d, a, b, f[e + 15], 14, 3634488961), b = l(b, c, d, a, f[e + 4], 20, 3889429448), a = l(a, b, c, d, f[e + 9], 5, 568446438), d = l(d, a, b, c, f[e + 14], 9, 3275163606), c = l(c, d, a, b, f[e + 3], 14, 4107603335), b = l(b, c, d, a, f[e + 8], 20, 1163531501), a = l(a, b, c, d, f[e + 13], 5, 2850285829), d = l(d, a, b, c, f[e + 2], 9, 4243563512), c = l(c, d, a, b, f[e + 7], 14, 1735328473), b = l(b, c, d, a, f[e + 12], 20, 2368359562), a = m(a, b, c, d, f[e + 5], 4, 4294588738), d = m(d, a, b, c, f[e + 8], 11, 2272392833), c = m(c, d, a, b, f[e + 11], 16, 1839030562), b = m(b, c, d, a, f[e + 14], 23, 4259657740), a = m(a, b, c, d, f[e + 1], 4, 2763975236), d = m(d, a, b, c, f[e + 4], 11, 1272893353), c = m(c, d, a, b, f[e + 7], 16, 4139469664), b = m(b, c, d, a, f[e + 10], 23, 3200236656), a = m(a, b, c, d, f[e + 13], 4, 681279174), d = m(d, a, b, c, f[e + 0], 11, 3936430074), c = m(c, d, a, b, f[e + 3], 16, 3572445317), b = m(b, c, d, a, f[e + 6], 23, 76029189), a = m(a, b, c, d, f[e + 9], 4, 3654602809), d = m(d, a, b, c, f[e + 12], 11, 3873151461), c = m(c, d, a, b, f[e + 15], 16, 530742520), b = m(b, c, d, a, f[e + 2], 23, 3299628645), a = n(a, b, c, d, f[e + 0], 6, 4096336452), d = n(d, a, b, c, f[e + 7], 10, 1126891415), c = n(c, d, a, b, f[e + 14], 15, 2878612391), b = n(b, c, d, a, f[e + 5], 21, 4237533241), a = n(a, b, c, d, f[e + 12], 6, 1700485571), d = n(d, a, b, c, f[e + 3], 10, 2399980690), c = n(c, d, a, b, f[e + 10], 15, 4293915773), b = n(b, c, d, a, f[e + 1], 21, 2240044497), a = n(a, b, c, d, f[e + 8], 6, 1873313359), d = n(d, a, b, c, f[e + 15], 10, 4264355552), c = n(c, d, a, b, f[e + 6], 15, 2734768916), b = n(b, c, d, a, f[e + 13], 21, 1309151649), a = n(a, b, c, d, f[e + 4], 6, 4149444226), d = n(d, a, b, c, f[e + 11], 10, 3174756917), c = n(c, d, a, b, f[e + 2], 15, 718787259), b = n(b, c, d, a, f[e + 9], 21, 3951481745), a = h(a, q), b = h(b, r), c = h(c, s), d = h(d, t);
    return (p(a) + p(b) + p(c) + p(d)).toLowerCase()
};
encodeConfig = function(d) {
    var txt = "";
    for (var n in d) {
        if (d.hasOwnProperty(n)) {
            var lines = String(d[n]).split("\n");
            txt += n + ": " + lines[0] + "\n";
            for (var i = 1; i < lines.length; i++) {
                txt += "    " + lines[i] + "\n";
            }
        }
    }

    return txt;
};
var parseConfig = function(txt) {
    txt = txt.split("\n");
    var k;
    var s;
    var o = {};
    for(var i=0; i<txt.length; i++) {
        if (txt[i] === "") continue;

        var c;
        c = -1;
        s = txt[i].split(/:[\s+]?(.+)?/, 2);
        k = s[0];
        if (s[1] === undefined)
            o[k] = "";
        else
            o[k] = s[1];

        if (i+1>=txt.length) continue;
        if (txt[i+1].substr(0,1) !== " ") continue;
        i++;

        c = c === -1 && txt[i].search(/\S/);
        for (;i<txt.length; i++) {
            o[k] += "\n" + txt[i].substr(c);

            if (i+1>=txt.length) break;
            if (txt[i+1].substr(0,1) !== " ") break;
        }
    }

    return o;
};

function pad(n, width, z) {
    z = z || '0';
    n = n + '';
    return n.length >= width ? n : new Array(width - n.length + 1).join(z) + n;
}

app.filter("txt", function() {
    return encodeConfig;
});
