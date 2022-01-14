<template>
    <div class="app">
        <div v-if="error" class="error">Error: {{error}}</div>
        <div class="category" v-for="(category, idx) in computedCategories" :key="idx">
            <div class="category-name">{{category.category}}</div>
            <div class="hosts">
                <div class="host" v-for="(host, idx) in category.hosts" :key="idx" :style="host | color">
                    <div class="host-name">{{host.host}}</div>
                    <div class="loading" v-show="host.ips.length === 0 && host.error == null"></div>
                    <div class="ips">
                        <div class="ip" v-for="(ip, idx) in host.ips" :key="idx">
                            <div class="ip-ip">{{ip.ip}}
                                <div class="loading" v-show="ip.latency == null"></div>
                                <div class="ip-latency" v-show="ip.latency != null && ip.error == null">{{ip.latency/1000}}ms</div>
                                <div class="ip-error" v-if="ip.error != null">No Response</div>
                            </div>
                        </div>
                    </div>
                    <div class="host-error" v-if="host.error">{{host.error}}</div>
                </div>
            </div>
            <hr v-if="idx !== categories.length - 1">
        </div>
    </div>
</template>
<script>
export default {
    data() {
        return {
            categories: [],
            hostsIdx: {},
            ipIdx: {},
            error: null,
        }
    },
    computed: {
        errors() {
            const errors = []
            for (const category of this.categories) {
                for (const host of category.hosts) {
                    if (host.error != null) {
                        errors.push(host)
                        continue
                    }
                    for (const ip of host.ips) {
                        if (ip.error != null) {
                            errors.push(host)
                            continue
                        }
                    }
                }
            }
            errors.sort((h1, h2) => h1.host.localeCompare(h2.host))
            return {category: "Errors", hosts: errors}
        },
        computedCategories() {
            const errors = this.errors
            if (errors.hosts.length === 0) {
                return this.categories
            }
            return ([errors]).concat(this.categories)
        },
    },
    filters: {
        color(host) {
            const loading = host.ips.filter(ip => ip.latency == null).length
            if ((host.error == null && host.ips.length === 0) || loading > 0) {
                return {backgroundColor: "#c9daf8"}
            }
            const down = host.ips.filter(ip => ip.error != null).length
            if (host.error != null || host.ips.length === down) {
                return {backgroundColor: "#f4cccc"}
            }
            if (down > 0) {
                return {backgroundColor: "#fce5cd"}
            }
            return {backgroundColor: "#b7e1cd"}
        },
    },
    async created() {
        let token
        try {
            token = await (await fetch("token")).text()
        } catch (e) {
            this.error = JSON.stringify(e)
            console.error({msg: "token error:", error: e})
            return
        }
        const socket = new WebSocket("ws://localhost/ws")

        socket.addEventListener("error", event => {
            this.error = JSON.stringify(event)
            console.error({msg: "websocket error:", error: event})
        })

        socket.addEventListener("open", () => {
            socket.send(JSON.stringify({token}))
        })

        socket.addEventListener("message", event => {
            const msg = JSON.parse(event.data)
            switch (msg.t) {
                case "s":
                    for (const category of msg.s) {
                        const c = {category: category.category, hosts: []}
                        this.categories.push(c)
                        for (const host of category.hosts) {
                            const h = {host, ips: [], error: null}
                            c.hosts.push(h)
                            if (host in this.hostsIdx) {
                                this.hostsIdx[host].push(h)
                            } else {
                                this.hostsIdx[host] = [h]
                            }
                        }
                    }
                    break
                case "r":
                    if (msg.i != null) {
                        for (const ip of msg.i) {
                            let i
                            if (!(ip in this.ipIdx)) {
                                let sortVal = 0
                                for (const [i, octet] of ip.split(".").entries()) {
                                    sortVal += (octet) << (3 - i)
                                }
                                i = {ip, latency: null, sortVal, error: null}
                                this.ipIdx[ip] = i
                            } else {
                                i = this.ipIdx[ip]
                            }

                            for (const host of this.hostsIdx[msg.h]) {
                                host.ips.push(i)
                            }
                        }
                        for (const host of this.hostsIdx[msg.h]) {
                            host.ips.sort((ip1, ip2) => ip1.sortVal - ip2.sortVal)
                        }
                    } else if (msg.e != null) {
                        for (const host of this.hostsIdx[msg.h]) {
                            host.error = msg.e
                        }
                    }
                    break
                case "p":
                    if (!(msg.i in this.ipIdx)) {
                        let sortVal = 0
                        for (const [i, octet] of msg.i.split(".").entries()) {
                            sortVal += (octet) << (3 - i)
                        }
                        this.ipIdx[msg.i] = {ip: msg.i, latency: msg.l, sortVal, error: msg.e}
                        return
                    }
                    this.ipIdx[msg.i].latency = msg.l
                    this.ipIdx[msg.i].error = msg.e
                    break
                case "c":
                    if (msg.e != null) {
                        this.error = msg.e
                    }
            }
        })
    },
}
</script>
<style lang="sass">
    .app
        width: 100%
        max-width: 1440px
        margin-left: auto
        margin-right: auto
        font-family: "Roboto"
        color: #222
        hr
            width: 95%
            border-top: 1px solid #888
            margin: 15px 0px 20px 0px
    .error
        font-size: 1.2em
        font-weight: bold
    .category
        width: 100%
        .category-name
            font-size: 1.6em
            font-weight: bold
            margin-bottom: 5px
        .hosts
            width: 100%
            display: grid
            grid-gap: 10px
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr))
            .host
                min-height: 75px
                padding: 10px
                .host-name
                    font-size: 1.2em
                    font-weight: bold
                .host-error
                    color: red
                .ip
                    padding: 5px
                    .ip-ip
                        font-weight: bold
                        display: flex
                        align-items: center
                        justify-content: left
                    .ip-latency, .ip-error
                        margin-left: 5px
                        display: inline
                        font-size: 0.8em
                        padding: 2px 5px
                        border-radius: 10px
                        background-color: rgba(0, 0, 0, 0.15)
                    .ip-error
                        background-color: #ff4444
                    .loading
                        margin-left: 5px

    .loading
        display: inline-block
        width: 16px
        height: 16px
        &:after
            content: " "
            display: block
            width: 16px
            height: 16px
            margin: 2px
            border-radius: 50%
            border: 1px solid #fff
            border-color: #000 transparent #000 transparent
            animation: loading 1.2s linear infinite

    @keyframes loading
        0%
            transform: rotate(0deg)
        100%
            transform: rotate(360deg)
</style>
