# frozen_string_literal: true

module Ledger
  class Account
    DEFAULT_CURRENCY = "EUR"

    attr_reader :owner, :currency, :entries

    def self.open(owner, currency: DEFAULT_CURRENCY)
      new(owner, currency).tap { |account| account.post(0, memo: "opening") }
    end

    def initialize(owner, currency)
      @owner = owner
      @currency = currency
      @entries = []
    end

    def post(amount_cents, memo: nil, kind: :credit)
      raise FrozenAccountError, "account #{owner} is frozen" if frozen_account?

      entry = Entry.new(amount_cents, kind, memo)
      entries << entry
      entry
    end

    def balance
      entries.sum(&:signed_amount)
    end
    alias total balance

    # def legacy_balance
    #   entries.inject(0) { |sum, entry| sum + entry.amount }
    # end

    class Entry
      attr_reader :amount, :kind, :memo

      def initialize(amount, kind, memo)
        @amount = amount
        @kind = kind
        @memo = memo
      end

      def debit?
        kind == :debit
      end

      def signed_amount
        debit? ? -amount : amount
      end
    end
  end
end
